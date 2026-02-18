package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/helpy/db"
	"github.com/psds-microservice/operator-directory-service/internal/config"
	"github.com/psds-microservice/operator-directory-service/internal/kafka"
	"github.com/psds-microservice/operator-directory-service/internal/model"
	"github.com/psds-microservice/operator-directory-service/internal/searchindex"
	"github.com/spf13/cobra"
)

var reindexSearchCmd = &cobra.Command{
	Use:   "reindex-search",
	Short: "Reindex all operators into search (Elasticsearch). Uses SEARCH_SERVICE_URL if set, otherwise skips (normal indexing via Kafka).",
	RunE:  runReindexSearch,
}

func init() {
	rootCmd.AddCommand(reindexSearchCmd)
}

func runReindexSearch(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	conn, err := db.Open(cfg.DSN())
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	var profiles []model.OperatorProfile
	if err := conn.Find(&profiles).Error; err != nil {
		return fmt.Errorf("list operators: %w", err)
	}
	log.Printf("reindex-search: found %d operators", len(profiles))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Приоритет: Kafka > HTTP
	if len(cfg.KafkaBrokers) > 0 && cfg.KafkaTopicOperator != "" {
		log.Println("reindex-search: using Kafka for reindexing")
		producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicOperator)
		defer producer.Close()
		for i := range profiles {
			producer.ProduceOperatorEvent(ctx, "operator.created", profiles[i].UserID, map[string]interface{}{
				"display_name": profiles[i].DisplayName,
				"region":       profiles[i].Region,
				"role":         profiles[i].Role,
			})
			if (i+1)%50 == 0 || i == len(profiles)-1 {
				log.Printf("reindex-search: sent %d/%d events to Kafka", i+1, len(profiles))
			}
		}
		log.Printf("reindex-search: done, sent %d events to Kafka (search-service worker will index them)", len(profiles))
		return nil
	}

	if cfg.SearchServiceURL != "" {
		log.Println("reindex-search: using HTTP for reindexing")
		client := searchindex.NewClient(cfg.SearchServiceURL)
		for i := range profiles {
			client.IndexOperator(ctx, &profiles[i])
			if (i+1)%50 == 0 || i == len(profiles)-1 {
				log.Printf("reindex-search: indexed %d/%d", i+1, len(profiles))
			}
		}
		log.Printf("reindex-search: done, indexed %d operators via HTTP", len(profiles))
		return nil
	}

	log.Println("reindex-search: neither KAFKA_BROKERS nor SEARCH_SERVICE_URL set")
	log.Println("reindex-search: normal indexing happens via Kafka events (search-service worker)")
	log.Printf("reindex-search: found %d operators (not reindexed)", len(profiles))
	return nil
}
