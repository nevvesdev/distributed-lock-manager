package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func newReleaseCmd() *cobra.Command {
	var owner string
	var token int64

	cmd := &cobra.Command{
		Use:     "release --key <chave> --owner <dono> --token <fencing-token>",
		Short:   "Libera um lock distribuído",
		Example: `  dlm release --key pagamento-123 --owner worker-A --token 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				return fmt.Errorf("flag --key é obrigatória")
			}
			if owner == "" {
				return fmt.Errorf("flag --owner é obrigatória")
			}
			if token == 0 {
				return fmt.Errorf("flag --token é obrigatória")
			}

			ctx := context.Background()

			client, err := infraredis.NewClient(ctx, infraredis.Config{Addr: redisAddr})
			if err != nil {
				return fmt.Errorf("falha ao conectar no Redis: %w", err)
			}

			repo := infraredis.NewLockRepository(client)
			svc := applock.NewLockService(repo)

			if err := svc.Release(ctx, key, owner, token); err != nil {
				fmt.Fprintf(os.Stderr, "erro: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("lock '%s' liberado com sucesso\n", key)
			return nil
		},
	}

	cmd.Flags().String("key", "", "chave do lock (obrigatório)")
	cmd.Flags().StringVar(&owner, "owner", "", "identificador do processo dono do lock (obrigatório)")
	cmd.Flags().Int64Var(&token, "token", 0, "fencing token retornado no acquire (obrigatório)")

	return cmd
}
