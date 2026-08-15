package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	domlock "github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status --key <chave>",
		Short:   "Consulta o estado atual de um lock",
		Example: `  dlm status --key pagamento-123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				return fmt.Errorf("flag --key é obrigatória")
			}

			ctx := context.Background()

			client, err := infraredis.NewClient(ctx, infraredis.Config{Addr: redisAddr})
			if err != nil {
				return fmt.Errorf("falha ao conectar no Redis: %w", err)
			}

			repo := infraredis.NewLockRepository(client)
			svc := applock.NewLockService(repo)

			l, err := svc.Get(ctx, key)
			if err != nil {
				if err == domlock.ErrLockNotFound {
					fmt.Printf("lock '%s' não existe ou já expirou\n", key)
					return nil
				}
				fmt.Fprintf(os.Stderr, "erro: %v\n", err)
				os.Exit(1)
			}

			out := map[string]any{
				"key":           l.Key,
				"owner":         l.Owner,
				"fencing_token": l.FencingToken,
				"ttl":           l.TTL.String(),
				"expires_at":    l.ExpiresAt.Format(time.RFC3339),
				"expirado":      l.IsExpired(),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}

	cmd.Flags().String("key", "", "chave do lock (obrigatório)")

	return cmd
}
