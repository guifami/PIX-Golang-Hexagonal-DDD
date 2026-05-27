# PIX API

Simulação completa do sistema de pagamentos **PIX** implementada em Go com arquitetura hexagonal, DDD e processamento assíncrono via Kafka.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-Web_Framework-008ECF?style=flat)](https://gin-gonic.com/)
[![Kafka](https://img.shields.io/badge/Apache_Kafka-3.7-231F20?style=flat&logo=apache-kafka)](https://kafka.apache.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Swagger](https://img.shields.io/badge/Swagger-OpenAPI-85EA2D?style=flat&logo=swagger)](http://localhost:8000/swagger/index.html)

---

## Visão Geral

Este projeto implementa os três pilares do sistema PIX:

- **Contas** — criação e gerenciamento com saldo em centavos (`int64`)
- **Chaves PIX** — registro de CPF, CNPJ, telefone, e-mail e EVP com validação por tipo
- **Pagamentos** — fluxo assíncrono completo: validação síncrona → enfileiramento Kafka → debit/crédito → notificação de resultado

O objetivo é demonstrar como implementar um domínio financeiro com regras de negócio reais (validação de CPF com dígitos verificadores, máquina de estados de transação, dinheiro em centavos) numa arquitetura que isola completamente a lógica de negócio de detalhes de infraestrutura.

---

## Sumário

- [Arquitetura](#arquitetura)
- [Fluxo de Pagamento](#fluxo-de-pagamento)
- [Pré-requisitos](#pré-requisitos)
- [Quick Start](#quick-start)
- [Desenvolvimento Local](#desenvolvimento-local)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Endpoints da API](#endpoints-da-api)
- [Regras de Negócio](#regras-de-negócio)
- [Segurança](#segurança)
- [Kafka](#kafka)
- [Banco de Dados](#banco-de-dados)
- [Testes](#testes)
- [Postman](#postman)
- [Estrutura do Projeto](#estrutura-do-projeto)
- [Comandos](#comandos)
- [Troubleshooting](#troubleshooting)

---

## Arquitetura

O projeto segue a **arquitetura hexagonal** (Ports & Adapters), onde a dependência flui estritamente para dentro — infraestrutura depende da aplicação, nunca o contrário.

```
┌─────────────────────────────────────────────────────────────────────┐
│                          ADAPTERS IN                                │
│                                                                     │
│   HTTP (Gin)                        Kafka Consumers                 │
│   ├─ AccountHandler                 ├─ PaymentConsumer              │
│   ├─ PixKeyHandler                  └─ ResultLogger                 │
│   └─ PaymentHandler                                                 │
│                                                                     │
│   Middleware: APIKeyAuth · RateLimit · SecurityHeaders              │
│               CORS · RequestTimeout · Idempotência                  │
└──────────────────────┬──────────────────────────────────────────────┘
                       │ Ports In (interfaces)
┌──────────────────────▼──────────────────────────────────────────────┐
│                       APPLICATION CORE                              │
│                                                                     │
│   AccountService      PixKeyService      PaymentService             │
│                                                                     │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                      DOMAIN                                 │   │
│   │  Account (CPF · Money)  ·  PixKey (KeyType)                 │   │
│   │  PixTransaction (PENDING → PROCESSING → COMPLETED/FAILED)   │   │
│   └─────────────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────────────┘
                       │ Ports Out (interfaces)
┌──────────────────────▼──────────────────────────────────────────────┐
│                         ADAPTERS OUT                                │
│                                                                     │
│   Postgres Repositories             Kafka Publisher                 │
│   ├─ AccountRepository              └─ KafkaEventPublisher          │
│   ├─ PixKeyRepository                                               │
│   └─ TransactionRepository                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Camadas

| Camada | Pacote | Responsabilidade |
|--------|--------|-----------------|
| **Domain** | `internal/domain/` | Entidades, value objects, erros de negócio. Zero dependências externas. |
| **Application** | `internal/application/` | Ports (interfaces) + use cases. Orquestra domínio e infraestrutura. |
| **Adapters In** | `internal/adapters/in/` | HTTP handlers (Gin) e Kafka consumers. Traduz protocolo → domínio. |
| **Adapters Out** | `internal/adapters/out/` | Repositórios Postgres e publisher Kafka. Traduz domínio → infraestrutura. |
| **Infrastructure** | `internal/infrastructure/` | Config, logger Zap, pool de conexões, factories Kafka. |
| **Entry Point** | `cmd/api/main.go` | Composition root: instancia tudo e inicia graceful shutdown. |

---

## Fluxo de Pagamento

O fluxo de pagamento é o coração do sistema. A resposta é imediata (202 Accepted) enquanto o processamento financeiro acontece de forma assíncrona via Kafka.

```
Cliente                 HTTP Handler              PaymentService            Kafka
  │                          │                         │                      │
  │  POST /pix/payments      │                         │                      │
  │─────────────────────────►│                         │                      │
  │                          │  Initiate()             │                      │
  │                          │────────────────────────►│                      │
  │                          │                         │ Valida payer (ACTIVE)│
  │                          │                         │ Valida receiver key  │
  │                          │                         │ Verifica saldo       │
  │                          │                         │ Cria tx: PENDING     │
  │                          │                         │ PublishInitiated()   │
  │                          │                         │─────────────────────►│
  │  202 Accepted            │                         │                      │
  │◄─────────────────────────│                         │                      │
  │  { status: "PENDING" }   │                                                │
  │                                                                           │
  │                    [assíncrono]          PaymentConsumer                  │
  │                                               │◄────────────────────────  │
  │                                               │  Process()               │
  │                                               │ Debita payer             │
  │                                               │ Credita receiver         │
  │                                               │ tx: COMPLETED            │
  │                                               │ PublishCompleted()───────►│
  │                                                                           │
  │                                          ResultLogger                    │
  │                                               │◄────────────────────────  │
  │                                               │  Loga resultado          │
  │                                                                           │
  │  GET /pix/payments/:id                                                    │
  │──────────────────────────────────────────────────────────────────────►   │
  │◄────────────── { status: "COMPLETED" } ─────────────────────────────     │
```

---

## Pré-requisitos

| Ferramenta | Versão | Finalidade |
|------------|--------|-----------|
| [Docker](https://docs.docker.com/get-docker/) + Compose | v2+ | Rodar toda a stack |
| [Go](https://go.dev/dl/) | 1.21+ | Desenvolvimento local |
| [swag CLI](https://github.com/swaggo/swag) | latest | Regenerar Swagger |

```bash
# Instalar swag CLI (apenas se for regenerar Swagger)
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## Quick Start

```bash
# Clona o projeto
git clone <repo-url>
cd golang-hexagonal

# Sobe toda a stack: Postgres + Kafka + Migrations + App
make up
```

Em segundos a API estará disponível:

| Serviço | URL |
|---------|-----|
| API | http://localhost:8000 |
| Health Check | http://localhost:8000/health |
| Swagger UI | http://localhost:8000/swagger/index.html |
| Kafka UI | http://localhost:8080 |

Para derrubar tudo:

```bash
make down
```

Para destruir todos os dados e reconstruir do zero:

```bash
make rebuild
```

---

## Desenvolvimento Local

Para rodar a aplicação Go diretamente na máquina (com Postgres e Kafka via Docker):

**1. Sobe apenas a infraestrutura:**

```bash
docker compose up go_db kafka kafka-ui migrate -d
```

**2. Cria o arquivo `.env.local`:**

```bash
APP_PORT=8000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=postgres
DB_SSLMODE=disable
KAFKA_BROKER=localhost:9092
KAFKA_GROUP_ID=pix-consumer-group
API_KEYS=
ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=60
```

**3. Roda a aplicação:**

```bash
make run
```

**Fluxo de onboarding completo (primeira vez):**

```bash
make setup    # tidy + swag + make up
```

---

## Variáveis de Ambiente

| Variável | Default (Docker) | Descrição |
|----------|-----------------|-----------|
| `APP_PORT` | `8000` | Porta HTTP da aplicação |
| `DB_HOST` | `go_db` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `postgres` | Usuário do banco |
| `DB_PASSWORD` | `1234` | Senha do banco |
| `DB_NAME` | `postgres` | Nome do banco |
| `DB_SSLMODE` | `disable` | Modo SSL (`disable`, `require`, `verify-full`) |
| `KAFKA_BROKER` | `kafka:9092` | Endereço do broker Kafka |
| `KAFKA_GROUP_ID` | `pix-consumer-group` | Consumer group do Kafka |
| `API_KEYS` | _(vazio)_ | Chaves separadas por vírgula. Vazio = auth desabilitada |
| `ALLOWED_ORIGINS` | `*` | Origins CORS permitidas, separadas por vírgula |
| `RATE_LIMIT_RPS` | `60` | Requisições por segundo por IP |

**Exemplo de configuração para produção:**

```bash
API_KEYS=chave-secreta-producao-1,chave-secreta-producao-2
ALLOWED_ORIGINS=https://meuapp.com,https://admin.meuapp.com
RATE_LIMIT_RPS=100
DB_SSLMODE=require
```

---

## Endpoints da API

> Todos os endpoints sob `/api/v1/` aceitam e retornam `application/json`.
> Quando `API_KEYS` está configurado, o header `X-API-Key` é obrigatório em todas as rotas `/api/v1/`.

---

### Contas

#### `POST /api/v1/accounts` — Criar conta

**Request:**
```json
{
  "owner_name": "João Silva",
  "cpf": "11144477735"
}
```

**Response `201 Created`:**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "owner_name": "João Silva",
  "cpf": "11144477735",
  "balance_cents": 0,
  "status": "ACTIVE",
  "created_at": "2025-05-26T10:00:00Z"
}
```

| Status | Motivo |
|--------|--------|
| `201` | Conta criada com sucesso |
| `400` | CPF inválido ou nome vazio |
| `409` | CPF já cadastrado |

---

#### `GET /api/v1/accounts/:id` — Buscar conta

**Response `200 OK`:**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "owner_name": "João Silva",
  "cpf": "11144477735",
  "balance_cents": 50000,
  "status": "ACTIVE",
  "created_at": "2025-05-26T10:00:00Z"
}
```

---

#### `GET /api/v1/accounts/:id/balance` — Consultar saldo

**Response `200 OK`:**
```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "balance_cents": 50000,
  "balance": "R$ 500.00"
}
```

---

#### `POST /api/v1/accounts/:id/deposit` — Depositar saldo

Suporta idempotência via header `X-Idempotency-Key`.

**Request:**
```json
{
  "amount_cents": 50000
}
```

**Response `200 OK`:** objeto da conta atualizado com o novo saldo.

| Status | Motivo |
|--------|--------|
| `200` | Depósito realizado |
| `400` | Valor inválido ou negativo |
| `404` | Conta não encontrada |

---

#### `GET /api/v1/accounts/:id/pix/keys` — Listar chaves da conta

**Response `200 OK`:**
```json
[
  {
    "id": "...",
    "account_id": "...",
    "key_type": "CPF",
    "key_value": "11144477735",
    "status": "ACTIVE",
    "created_at": "2025-05-26T10:00:00Z"
  }
]
```

---

#### `GET /api/v1/accounts/:id/pix/payments` — Listar pagamentos da conta

Retorna todas as transações onde a conta é pagadora.

**Response `200 OK`:** array de transações (mesmo formato de `GET /api/v1/pix/payments/:id`).

---

### Chaves PIX

#### `POST /api/v1/pix/keys` — Registrar chave

**Request:**
```json
{
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "key_type": "EMAIL",
  "key_value": "joao@email.com"
}
```

**Response `201 Created`:**
```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "key_type": "EMAIL",
  "key_value": "joao@email.com",
  "status": "ACTIVE",
  "created_at": "2025-05-26T10:05:00Z"
}
```

| Status | Motivo |
|--------|--------|
| `201` | Chave registrada |
| `400` | Formato inválido para o tipo de chave |
| `404` | Conta não encontrada |
| `409` | Chave já cadastrada |
| `422` | Limite de 5 chaves atingido ou conta inativa |

---

#### `GET /api/v1/pix/keys/:key` — Buscar chave por valor

```
GET /api/v1/pix/keys/joao@email.com
GET /api/v1/pix/keys/11144477735
GET /api/v1/pix/keys/+5511999990000
```

**Response `200 OK`:** objeto da chave (mesmo formato acima).

---

#### `DELETE /api/v1/pix/keys/:key?account_id=` — Deletar chave

Requer `account_id` como query parameter para verificação de propriedade.

```
DELETE /api/v1/pix/keys/joao@email.com?account_id=a1b2c3d4-...
```

| Status | Motivo |
|--------|--------|
| `204` | Chave deletada |
| `403` | A chave não pertence à conta informada |
| `404` | Chave não encontrada |

---

### Pagamentos

#### `POST /api/v1/pix/payments` — Iniciar pagamento

Suporta idempotência via header `X-Idempotency-Key`. Responde `202 Accepted` imediatamente — o processamento é assíncrono.

**Request:**
```json
{
  "payer_account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "receiver_key": "joao@email.com",
  "amount_cents": 15000,
  "description": "Aluguel de maio"
}
```

**Response `202 Accepted`:**
```json
{
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "payer_account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "receiver_key": "joao@email.com",
  "receiver_account_id": null,
  "amount_cents": 15000,
  "status": "PENDING",
  "description": "Aluguel de maio",
  "initiated_at": "2025-05-26T10:10:00Z",
  "completed_at": null,
  "failure_reason": null
}
```

| Status | Motivo |
|--------|--------|
| `202` | Pagamento aceito, processando |
| `400` | Valor inválido ou pagador = recebedor |
| `404` | Conta ou chave não encontrada |
| `422` | Conta inativa, saldo insuficiente |

---

#### `GET /api/v1/pix/payments/:id` — Consultar transação

Consulte após o `POST` para verificar o resultado do processamento assíncrono.

**Response `200 OK` (após processado):**
```json
{
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "payer_account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "receiver_key": "joao@email.com",
  "receiver_account_id": "d4e5f6a7-b8c9-0123-defa-234567890123",
  "amount_cents": 15000,
  "status": "COMPLETED",
  "description": "Aluguel de maio",
  "initiated_at": "2025-05-26T10:10:00Z",
  "completed_at": "2025-05-26T10:10:01Z",
  "failure_reason": null
}
```

---

### Sistema

#### `GET /health` — Status da API

Não requer autenticação.

**Response `200 OK`:**
```json
{
  "status": "ok",
  "database": "up"
}
```

**Response `503 Service Unavailable`:**
```json
{
  "status": "error",
  "database": "down"
}
```

#### `GET /swagger/*` — Documentação interativa

Swagger UI disponível em `http://localhost:8000/swagger/index.html`.

---

## Regras de Negócio

### CPF

- Deve ter exatamente 11 dígitos numéricos
- Rejeita sequências homogêneas (ex: `11111111111`, `00000000000`)
- Valida os **dois dígitos verificadores** pelo algoritmo módulo 11
- Armazenado sem formatação (apenas dígitos)

### Dinheiro (`Money`)

- Representado em **centavos** como `int64` — nunca `float64`
- A API aceita e retorna `amount_cents` (inteiro)
- Valor negativo é rejeitado no construtor
- `Money.String()` formata como `"R$ X.YY"` (apenas para exibição)

### Chaves PIX

| Tipo | Formato | Regex |
|------|---------|-------|
| `CPF` | 11 dígitos | `^\d{11}$` |
| `CNPJ` | 14 dígitos | `^\d{14}$` |
| `PHONE` | `+` + DDI + número | `^\+\d{10,13}$` |
| `EMAIL` | Endereço válido | RFC pattern, case-insensitive |
| `EVP` | UUID v4 | UUID pattern, case-insensitive |

- **Máximo de 5 chaves** por conta
- Cada valor de chave é **único** globalmente
- Apenas contas `ACTIVE` podem registrar chaves

### Transações

A transação segue uma máquina de estados estrita:

```
PENDING ──→ PROCESSING ──→ COMPLETED
                      └──→ FAILED
```

| Estado | Descrição |
|--------|-----------|
| `PENDING` | Criada após validação síncrona. Publicada no Kafka. |
| `PROCESSING` | Consumer Kafka iniciou o processamento (idempotente). |
| `COMPLETED` | Débito e crédito aplicados. `receiver_account_id` preenchido. |
| `FAILED` | Erro durante processamento. `failure_reason` preenchido. |

Transições inválidas (ex: `COMPLETED → PENDING`) lançam `ErrInvalidTransactionStatus`.

### Pagamento — Validações síncronas

Antes de publicar no Kafka, `PaymentService.Initiate` valida:

1. Conta do pagador existe e está `ACTIVE`
2. Chave do recebedor existe
3. Pagador ≠ Recebedor (mesma conta)
4. Saldo do pagador ≥ valor da transação

### Pagamento — Processamento assíncrono

Ao consumir de `pix.payment.initiated`, `PaymentService.Process`:

1. Revalida existência da chave do recebedor
2. Busca contas do pagador e recebedor
3. Debita o pagador
4. Credita o recebedor
5. Persiste ambas as contas
6. Marca transação como `COMPLETED`
7. Publica `pix.payment.completed`

Em caso de erro em qualquer etapa: marca como `FAILED` e publica `pix.payment.failed`.

---

## Segurança

Todos os endpoints `/api/v1/` passam pela seguinte cadeia de middlewares:

```
Request → APIKeyAuth → RateLimiter → SecurityHeaders → CORS → RequestTimeout → Handler
```

### API Key

- Header: `X-API-Key`
- Configurar via `API_KEYS` (comma-separated). Vazio = desabilitado.
- Retorna `401 Unauthorized` se ausente ou inválida.

### Rate Limiting

- Algoritmo: **token bucket** por IP
- Default: 60 req/s por IP (configurável via `RATE_LIMIT_RPS`)
- Retorna `429 Too Many Requests` quando excedido
- Cleanup automático de entradas inativas (10+ min)

### Security Headers

Todos os responses incluem:

| Header | Valor |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Cache-Control` | `no-store` |
| `Content-Security-Policy` | `default-src 'none'` (exceto `/swagger/`) |

### Idempotência

- Header: `X-Idempotency-Key` (opcional)
- Rotas: `POST /pix/payments` e `POST /accounts/:id/deposit`
- TTL: 24 horas (in-memory)
- Resposta repetida inclui header `X-Idempotency-Replayed: true`

### Outros

- **CORS**: origins configuráveis via `ALLOWED_ORIGINS`. Métodos: GET, POST, DELETE, OPTIONS.
- **Request Timeout**: 30s nas rotas `/api/v1/`
- **Panic Recovery**: qualquer panic retorna `500` estruturado e é logado via Zap

---

## Kafka

### Tópicos

| Tópico | Publicado por | Consumido por | Quando |
|--------|--------------|---------------|--------|
| `pix.payment.initiated` | `PaymentService.Initiate` | `PaymentConsumer` | Transação criada como PENDING |
| `pix.payment.completed` | `PaymentService.Process` | `ResultLogger` | Processamento concluído |
| `pix.payment.failed` | `PaymentService.Process` | `ResultLogger` | Erro no processamento |

### Formato da Mensagem

Todos os tópicos usam o mesmo schema `PaymentEvent`:

```json
{
  "transaction_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "payer_account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "receiver_key": "joao@email.com",
  "amount_cents": 15000,
  "status": "PENDING",
  "occurred_at": "2025-05-26T10:10:00Z",
  "failure_reason": null
}
```

### Kafka UI

Acesse `http://localhost:8080` para inspecionar tópicos, mensagens e consumer groups em tempo real.

---

## Banco de Dados

### Schema

```
accounts ──1:N── pix_keys
    │
    └──1:N── pix_transactions (como payer_account_id)
    └──1:N── pix_transactions (como receiver_account_id)
```

**Tabela `accounts`:**

| Coluna | Tipo | Constraint |
|--------|------|-----------|
| `id` | UUID | PK |
| `owner_name` | VARCHAR(255) | NOT NULL |
| `cpf` | VARCHAR(11) | UNIQUE, NOT NULL |
| `balance_cents` | BIGINT | DEFAULT 0 |
| `status` | VARCHAR(20) | DEFAULT 'ACTIVE' |
| `created_at` | TIMESTAMPTZ | NOT NULL |
| `updated_at` | TIMESTAMPTZ | NOT NULL |

**Tabela `pix_keys`:**

| Coluna | Tipo | Constraint |
|--------|------|-----------|
| `id` | UUID | PK |
| `account_id` | UUID | FK → accounts ON DELETE CASCADE |
| `key_type` | VARCHAR(20) | NOT NULL |
| `key_value` | VARCHAR(255) | UNIQUE, NOT NULL |
| `status` | VARCHAR(20) | DEFAULT 'ACTIVE' |
| `created_at` | TIMESTAMPTZ | NOT NULL |

**Tabela `pix_transactions`:**

| Coluna | Tipo | Constraint |
|--------|------|-----------|
| `id` | UUID | PK |
| `payer_account_id` | UUID | FK → accounts |
| `receiver_key` | VARCHAR(255) | NOT NULL |
| `receiver_account_id` | UUID | FK → accounts, nullable |
| `amount_cents` | BIGINT | CHECK > 0 |
| `status` | VARCHAR(20) | DEFAULT 'PENDING' |
| `description` | VARCHAR(140) | nullable |
| `initiated_at` | TIMESTAMPTZ | NOT NULL |
| `completed_at` | TIMESTAMPTZ | nullable |
| `failure_reason` | VARCHAR(255) | nullable |

### Migrations

As migrations são aplicadas automaticamente pelo serviço `migrate` no `docker compose up`. Numeradas sequencialmente em `migrations/`.

Para aplicar manualmente:

```bash
docker compose run --rm migrate
```

---

## Testes

```bash
make test             # todos os testes
make cover            # relatório de cobertura (domain + usecase)
go test ./internal/application/usecase/...  # apenas use cases
```

### Cobertura atual

| Pacote | Cobertura |
|--------|-----------|
| `internal/domain/account` | 96.0% |
| `internal/domain/pixkey` | 100.0% |
| `internal/domain/transaction` | 100.0% |
| `internal/application/usecase` | 97.1% |

Os testes utilizam **mocks in-memory** nos pacotes `_test` — sem dependência de banco real ou Kafka, garantindo testes rápidos e determinísticos.

---

## Postman

Importe os dois arquivos do diretório `postman/`:

1. `postman/pix-api.postman_collection.json` — collection completa
2. `postman/pix-local.postman_environment.json` — environment local

### Fluxo recomendado

Execute as pastas na ordem:

| Pasta | O que faz |
|-------|-----------|
| **1. Setup** | Cria duas contas e deposita saldo. Preenche `payer_account_id` e `receiver_account_id`. |
| **2. Contas** | CRUD completo de contas e consulta de saldo. |
| **3. Chaves PIX** | Registra chave para o recebedor. Preenche `pix_key`. |
| **4. Pagamentos PIX** | Inicia pagamento → aguarda Kafka processar → verifica status COMPLETED. Preenche `transaction_id`. |
| **5. Cenários de Erro** | Valida todos os erros: saldo insuficiente, chave duplicada, 401, 429, etc. |

> Todos os IDs são propagados automaticamente via scripts de teste do Postman. Configure a variável `api_key` no environment se `API_KEYS` estiver ativo no servidor.

---

## Estrutura do Projeto

```
.
├── cmd/
│   └── api/
│       └── main.go                  # Composition root + graceful shutdown
├── internal/
│   ├── domain/                      # Lógica de negócio pura (zero deps externas)
│   │   ├── account/
│   │   │   ├── entity.go            # Account: Debit, Credit, Activate, Deactivate
│   │   │   ├── value_objects.go     # Money (int64 cents), CPF (dígitos verificadores)
│   │   │   └── errors.go            # Erros de domínio (ErrInsufficientFunds, etc.)
│   │   ├── pixkey/
│   │   │   ├── entity.go            # PixKey: verificação de propriedade
│   │   │   ├── value_objects.go     # KeyType, KeyStatus, validação por regex
│   │   │   └── errors.go
│   │   └── transaction/
│   │       ├── entity.go            # PixTransaction: máquina de estados
│   │       ├── value_objects.go     # TransactionStatus
│   │       └── errors.go
│   ├── application/
│   │   ├── ports/
│   │   │   ├── in/                  # Interfaces dos use cases (entrada)
│   │   │   └── out/                 # Interfaces dos repositórios e publisher (saída)
│   │   └── usecase/
│   │       ├── account_service.go
│   │       ├── pixkey_service.go
│   │       └── payment_service.go
│   ├── adapters/
│   │   ├── in/
│   │   │   ├── httpadapter/
│   │   │   │   ├── router.go        # Setup de rotas e middlewares
│   │   │   │   ├── account_handler.go
│   │   │   │   ├── pixkey_handler.go
│   │   │   │   ├── payment_handler.go
│   │   │   │   ├── middleware.go    # Request logging com request_id
│   │   │   │   ├── security.go     # APIKey, RateLimit, Headers, CORS, Timeout
│   │   │   │   ├── idempotency.go  # In-memory idempotency store
│   │   │   │   ├── health.go
│   │   │   │   └── response.go     # ErrorResponse struct
│   │   │   └── kafka/
│   │   │       ├── payment_consumer.go
│   │   │       └── result_logger.go
│   │   └── out/
│   │       ├── postgres/
│   │       │   ├── account_repository.go
│   │       │   ├── pixkey_repository.go
│   │       │   └── transaction_repository.go
│   │       └── kafka/
│   │           ├── publisher.go
│   │           └── event.go
│   └── infrastructure/
│       ├── config/config.go         # Leitura de env vars com defaults
│       ├── database/postgres.go     # Pool de conexões (25 max/idle, 5m lifetime)
│       ├── logger/                  # Zap global + propagação via context
│       └── kafka/                   # Factories writer/reader + EnsureTopics
├── migrations/                      # SQL ordenado (migrate/migrate)
├── docs/                            # Swagger gerado — não editar manualmente
├── postman/                         # Collection + environment Postman
├── docker-compose.yml
├── Dockerfile                       # Multi-stage build
├── Makefile
├── go.mod
└── .env / .env.local
```

---

## Comandos

```bash
make up          # Sobe toda a stack (Postgres + Kafka + Migrations + App)
make down        # Derruba os serviços
make rebuild     # Destrói volumes e reconstrói tudo do zero
make run         # Roda localmente com .env.local
make setup       # Onboarding completo: tidy + swag + up

make swag        # Regenera documentação Swagger (necessário após mudar annotations)
make fmt         # go fmt ./...
make tidy        # go mod tidy
make test        # go test ./internal/...
make cover       # Relatório de cobertura de domain + usecase
```

---

## Troubleshooting

### App não conecta no banco na primeira subida

O serviço `migrate` precisa terminar antes do app iniciar. O Docker Compose aguarda via `depends_on + condition: service_completed_successfully`. Se ainda ocorrer, aguarde alguns segundos e rode:

```bash
docker compose restart go_app
```

### Kafka: mensagens não sendo consumidas

Verifique se os tópicos foram criados:

```bash
# Acesse o Kafka UI em http://localhost:8080 → Topics
# Ou via CLI:
docker compose exec kafka kafka-topics.sh --bootstrap-server localhost:9092 --list
```

O app cria os tópicos automaticamente via `EnsureTopics` na inicialização, mas o broker precisa estar disponível.

### Pagamento fica em PENDING

O consumer Kafka (`PaymentConsumer`) pode não estar rodando. Verifique os logs:

```bash
docker compose logs go_app | grep -i "consumer\|kafka\|error"
```

### Migrations falham

Verifique se o Postgres está saudável antes das migrations rodarem:

```bash
docker compose logs migrate
docker compose logs go_db
```

Para recriar o banco do zero:

```bash
make rebuild
```

### Regenerar Swagger

Após modificar qualquer annotation (`@Summary`, `@Param`, `@Success`, `@Router`):

```bash
make swag
```

O diretório `docs/` é gerado automaticamente — nunca edite manualmente.

---

## Stack Tecnológica

| Componente | Tecnologia |
|------------|-----------|
| Linguagem | Go 1.21 |
| Web Framework | [Gin](https://gin-gonic.com/) |
| Banco de Dados | PostgreSQL 16 |
| Mensageria | Apache Kafka 3.7 + [kafka-go](https://github.com/segmentio/kafka-go) |
| Logging | [Uber Zap](https://github.com/uber-go/zap) |
| API Docs | [swaggo/swag](https://github.com/swaggo/swag) (OpenAPI 2.0) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Driver SQL | [lib/pq](https://github.com/lib/pq) (raw `database/sql`) |
| UUID | [google/uuid](https://github.com/google/uuid) |
| Rate Limiting | `golang.org/x/time/rate` (token bucket) |
| Containerização | Docker + Docker Compose |
