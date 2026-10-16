# Leilões com fechamento automático (Go + Goroutines)

Sistema de leilões do curso Go Expert, com a funcionalidade de **fechamento automático**: ao criar um leilão, uma goroutine é agendada para encerrá-lo (status `Completed`) assim que a duração configurada expirar, sem intervenção manual.

A implementação do fechamento está em `internal/infra/database/auction/create_auction.go`.

## Como funciona o fechamento

- Ao persistir um leilão, `CreateAuction` dispara uma goroutine (`scheduleAuctionClosing`) que não bloqueia a thread principal.
- A goroutine calcula o instante de fechamento (`timestamp do leilão + duração`) e, ao expirar, faz um `UpdateOne` no MongoDB alterando o status para `Completed`.
- O update é condicional (`status = Active`), então um leilão já encerrado não é reprocessado.
- A duração é lida de variáveis de ambiente, reaproveitando o mesmo intervalo usado pela validação de lances (bids), para que o corte de lances e o fechamento fiquem coerentes.

## Variáveis de ambiente (tempo)

| Variável | Descrição | Exemplo |
|---|---|---|
| `AUCTION_DURATION` | Duração do leilão até o fechamento automático (tem precedência) | `20s`, `1m`, `2h` |
| `AUCTION_INTERVAL` | Usado se `AUCTION_DURATION` não estiver definida; também controla o corte de lances | `20s` |

O valor aceita qualquer formato de `time.ParseDuration` (`30s`, `5m`, `1h`...). Se nenhuma estiver definida, o padrão é `5m`.

Demais variáveis (já no `docker-compose.yml`): `MONGODB_URL`, `MONGODB_DB`, `BATCH_INSERT_INTERVAL`, `MAX_BATCH_SIZE`.

> As variáveis são fornecidas pelo `docker-compose.yml` (seção `environment`). Para rodar fora do Docker, copie `cmd/auction/.env.example` para `cmd/auction/.env`.

## Como rodar (Docker Compose)

```bash
docker compose up --build
```

Sobe o MongoDB e a aplicação (porta `8080`). Nenhum passo manual adicional é necessário.

### Testar o fechamento automático via API

```bash
# 1. Criar um leilão
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{"product_name":"Playstation 5","category":"Games","description":"Console novo lacrado na caixa","condition":1}'

# 2. Descobrir o id de um leilão aberto (status 0 = Active)
curl "http://localhost:8080/auction?status=0"

# 3. Consultar o leilão (logo após criar: status 0)
curl "http://localhost:8080/auction/<ID>"

# 4. Após AUCTION_DURATION, consultar de novo: status 1 = Completed (fechado automaticamente)
curl "http://localhost:8080/auction/<ID>"
```

## Teste automatizado

O teste `TestAuctionIsAutomaticallyClosedAfterDuration` (em `internal/infra/database/auction/create_auction_test.go`) comprova o fechamento:

1. Cria um leilão → status `Active`.
2. Aguarda o tempo configurado em `AUCTION_DURATION`.
3. Verifica que o status mudou automaticamente para `Completed`.

Ele sobe um MongoDB efêmero via [Testcontainers](https://testcontainers.com/) — basta ter o Docker em execução:

```bash
go test ./...
```

## Principais rotas

| Método | Rota | Descrição |
|---|---|---|
| POST | `/auction` | Cria um leilão |
| GET | `/auction?status=0` | Lista leilões (0 = abertos, 1 = fechados) |
| GET | `/auction/:auctionId` | Detalha um leilão |
| GET | `/auction/winner/:auctionId` | Lance vencedor |
| POST | `/bid` | Cria um lance |
| GET | `/bid/:auctionId` | Lista lances de um leilão |
| GET | `/user/:userId` | Detalha um usuário |

## Status do leilão

`Active` = `0` (aberto) · `Completed` = `1` (fechado).
