## Serviço de Gerenciamento de Pedidos (Order Service)

Microsserviço em Go para gerenciamento de pedidos de e-commerce, seguindo arquitetura hexagonal, com MongoDB, RabbitMQ, Gin, testes (unitários, integração e end-to-end), documentação Swagger.

![Modelagem do banco](Modelagem%20do%20banco.drawio.png)

### Stack utilizada

- **Linguagem**: Go 1.26
- **HTTP**: Gin
- **Banco de dados**: MongoDB
- **Mensageria**: RabbitMQ (AMQP)
- **Logger**: Zap
- **Configuração**: Cleanenv
- **Testes**: Testify, Dockertest
- **Docs API**: Swag + gin-swagger
- **Containerização**: Docker + Docker Compose

### Instruções de execução

- Copie `.env.example` para `.env` e ajuste os valores se necessário.
- Suba toda a stack (app + MongoDB + RabbitMQ) com:

```bash
docker-compose up --build -d
```

ou

```bash
make docker-up
```

- Derrubar a stack:

```bash
make docker-down
```

- Desenvolvimento com hot-reload (requer `air` instalado via `go tool`):

```bash
make run
```

### Variáveis de ambiente

As configurações são carregadas via `cleanenv` a partir do arquivo `.env`.

| Variável                 | Obrigatória | Default      | Descrição                              |
|--------------------------|------------|--------------|----------------------------------------|
| `APP_PORT`              | Não        | `8080`       | Porta HTTP da aplicação                |
| `APP_ENV`               | Não        | `development`| Ambiente (`development` \| `production`)| 
| `APP_READ_TIMEOUT_SECONDS`  | Não    | `15`         | Timeout de leitura HTTP (segundos)     |
| `APP_WRITE_TIMEOUT_SECONDS` | Não    | `15`         | Timeout de escrita HTTP (segundos)     |
| `APP_IDLE_TIMEOUT_SECONDS`  | Não    | `60`         | Timeout de conexões ociosas HTTP (segundos) |
| `MONGODB_URI`           | **Sim**    | —            | URI de conexão do MongoDB             |
| `MONGODB_DATABASE`      | **Sim**    | —            | Nome do database do MongoDB           |
| `MONGODB_TIMEOUT_SECONDS` | Não      | `10`         | Timeout (segundos) para operações Mongo|
| `RABBITMQ_URI`          | **Sim**    | —            | URI de conexão do RabbitMQ            |
| `RABBITMQ_EXCHANGE`     | Não        | `orders`     | Nome do exchange para eventos de pedido|
| `RABBITMQ_QUEUE`        | **Sim**    | —            | Nome da fila de eventos de status     |
| `RABBITMQ_ROUTING_KEY`  | **Sim**    | —            | Routing key para eventos de status    |
| `LOGGER_LEVEL`          | Não        | `info`       | Nível de log (`debug`, `info`, etc.)  |

### Comandos principais (Makefile)

- **Rodar em desenvolvimento (hot reload)**: `make run`
- **Build do binário**: `make build`
- **Testes + cobertura (domínio, aplicação, adapters)**: `make test`
- **Cobertura (resumo no terminal)**: `make coverage`
- **Cobertura em HTML (`coverage.html`)**: `make coverage-html`
- **Gerar documentação Swagger**: `make swagger`
- **Lint**: `make lint`
- **Subir stack com Docker Compose**: `make docker-up`
- **Derrubar stack**: `make docker-down`

### Endpoints principais

- `POST /orders` – cria pedido
- `GET /orders/:id` – consulta pedido por ID
- `PATCH /orders/:id/status` – atualiza status do pedido (publica evento no RabbitMQ). Aceita qualquer transição entre os status válidos (`criado`, `em_processamento`, `enviado`, `entregue`), mas rejeita tentativas de atualizar para o mesmo status atual.
- `GET /health` – healthcheck
- `GET /swagger/index.html` – Swagger UI (após `make swagger` e app rodando)

### Testes e cobertura

- **Unitários**:
  - Camadas `internal/domain` e `internal/application`.
  - Casos: criação de pedido, validações, transições de status, erros de repositório/publisher.
- **Integração**:
  - Pasta `tests/` com suites usando Dockertest para:
    - MongoDB (`order_repository_integration_test.go`)
    - RabbitMQ (`publisher_integration_test.go`)
    - Fluxo end-to-end HTTP → MongoDB → RabbitMQ (`e2e_test.go`)
- **Cobertura**:
  - Rodar todos os testes com cobertura focada em domínio, aplicação e adapters:

```bash
make test
```

  Ou, sem Make:

```bash
go test ./... -coverpkg=./internal/domain/...,./internal/application/...,./internal/adapters/http/...,./internal/adapters/mongodb/...,./internal/adapters/rabbitmq/... -coverprofile=coverage.out
```

  - Ver resumo de cobertura por função no terminal:

```bash
make coverage
```

  Ou, sem Make (execute `make test` ou o comando acima antes, para gerar `coverage.out`):

```bash
go tool cover -func=coverage.out
```

  - Gerar relatório HTML em `coverage.html` (meta ≥ 60%):

```bash
make coverage-html
```

  Ou, sem Make:

```bash
go tool cover -html=coverage.out -o coverage.html
```

### Principais decisões técnicas

- **Arquitetura**: hexagonal (Ports & Adapters), com domínio e aplicação independentes de detalhes de infra.
- **Persistência**: MongoDB Driver v2, repositório recebe `*mongo.Database` para facilitar troca de banco e testes.
- **Mensageria**: RabbitMQ com exchange `orders`, routing key `order.status.updated` e evento de mudança de status tipado.
- **Configuração**: `cleanenv` lendo exclusivamente de `.env`, com validação de variáveis obrigatórias.
- **Observabilidade**: logs estruturados com Zap, middleware adicionando `trace_id` e inclusão de `order_id` em operações críticas.
- **Testes**: uso de `testify/mock` para mocks de ports, Dockertest para MongoDB/RabbitMQ e suite e2e exercendo o fluxo completo da API.