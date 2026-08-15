package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func newWatchCmd() *cobra.Command {
	var owner string
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:     "watch --key <chave> --owner <dono> --ttl <duração>",
		Short:   "Adquire um lock e mantém via heartbeat até Ctrl+C",
		Example: `  dlm watch --key pagamento-123 --owner worker-A --ttl 30s`,
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

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			client, err := infraredis.NewClient(ctx, infraredis.Config{Addr: redisAddr})
			if err != nil {
				return fmt.Errorf("falha ao conectar no Redis: %w", err)
			}

			repo := infraredis.NewLockRepository(client)
			svc := applock.NewLockService(repo)

			handle, err := svc.AcquireWithHeartbeat(ctx, key, owner, ttl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "erro ao adquirir lock: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("lock adquirido — key=%s owner=%s token=%d ttl=%s\n",
				key, owner, handle.Lock.FencingToken, ttl)
			fmt.Println("heartbeat ativo — pressione Ctrl+C para liberar o lock")

			<-ctx.Done()

			fmt.Println("\nliberando lock...")
			releaseCtx := context.Background()
			if err := handle.Release(releaseCtx); err != nil {
				fmt.Fprintf(os.Stderr, "erro ao liberar lock: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("lock '%s' liberado com sucesso\n", key)
			return nil
		},
	}

	cmd.Flags().String("key", "", "chave do lock (obrigatório)")
	cmd.Flags().StringVar(&owner, "owner", "", "identificador do processo dono do lock (obrigatório)")
	cmd.Flags().DurationVar(&ttl, "ttl", 30*time.Second, "tempo de vida do lock com renovação automática")

	return cmd
}
