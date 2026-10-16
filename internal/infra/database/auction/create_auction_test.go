package auction

import (
	"context"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupMongo(t *testing.T) *mongo.Database {
	t.Helper()
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:6")
	if err != nil {
		t.Fatalf("erro ao subir container do mongo: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("erro ao obter connection string: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("erro ao conectar no mongo: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(ctx)
	})

	return client.Database("auctions_test")
}

func TestAuctionIsAutomaticallyClosedAfterDuration(t *testing.T) {
	db := setupMongo(t)

	t.Setenv("AUCTION_DURATION", "2s")

	repo := NewAuctionRepository(db)

	auction, errEntity := auction_entity.CreateAuction(
		"Playstation 5",
		"Games",
		"Console Playstation 5 em ótimo estado",
		auction_entity.New,
	)
	if errEntity != nil {
		t.Fatalf("erro ao criar entidade de leilão: %v", errEntity.Message)
	}

	if errCreate := repo.CreateAuction(context.Background(), auction); errCreate != nil {
		t.Fatalf("erro ao persistir leilão: %v", errCreate.Message)
	}

	created, errFind := repo.FindAuctionById(context.Background(), auction.Id)
	if errFind != nil {
		t.Fatalf("erro ao buscar leilão recém-criado: %v", errFind.Message)
	}
	if created.Status != auction_entity.Active {
		t.Fatalf("leilão deveria iniciar aberto (Active), status = %d", created.Status)
	}

	time.Sleep(3 * time.Second)

	closed, errFind := repo.FindAuctionById(context.Background(), auction.Id)
	if errFind != nil {
		t.Fatalf("erro ao buscar leilão após expiração: %v", errFind.Message)
	}
	if closed.Status != auction_entity.Completed {
		t.Fatalf("leilão deveria ter sido fechado automaticamente (Completed), status = %d", closed.Status)
	}
}
