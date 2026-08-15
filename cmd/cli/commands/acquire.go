package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func newAcquireCmd() *cobra.Command {
	var owner string
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "acquire --key <chave> --owner <dono> --ttl <duração>",
		Short: "Adquire um lock distribuído",
		Example: `  dlm acquire --key pagamento-123 --owner worker-A --ttl 30s
  dlm acquire --key pedido-456 --owner worker-B --ttl 1m --redis localhost:6380`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				return fmt.Errorf("flag --key é obrigatória")
			}
			if owner == "" {
				return fmt.Errorf("flag --owner é obrigatória")
			}
			if ttl <= 0 {
				return fmt.Errorf("flag --ttl deve ser maior que zero (ex: 30s, 1m)")
			}

			ctx := context.Background()

			client, err := infraredis.NewClient(ctx, infraredis.Config{Addr: redisAddr})
			if err != nil {
				return fmt.Errorf("falha ao conectar no Redis: %w", err)
			}

			repo := infraredis.NewLockRepository(client)
			svc := applock.NewLockService(repo)

			l, err := svc.Acquire(ctx, key, owner, ttl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "erro: %v\n", err)
				os.Exit(1)
			}

			out := map[string]any{
				"key":           l.Key,
				"owner":         l.Owner,
				"fencing_token": l.FencingToken,
				"ttl":           l.TTL.String(),
				"expires_at":    l.ExpiresAt.Format(time.RFC3339),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}

	cmd.Flags().String("key", "", "chave do lock (obrigatório)")
	cmd.Flags().StringVar(&owner, "owner", "", "identificador do processo dono do lock (obrigatório)")
	cmd.Flags().DurationVar(&ttl, "ttl", 30*time.Second, "tempo de vida do lock (ex: 30s, 1m)")

	return cmd
}
