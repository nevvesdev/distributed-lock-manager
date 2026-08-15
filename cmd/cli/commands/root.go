package commands

import (
	"github.com/spf13/cobra"
)

var redisAddr string

// NewRootCmd cria o comando raiz do CLI do distributed-lock-manager.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "dlm",
		Short: "distributed-lock-manager — CLI para operações de lock distribuído",
		Long: `dlm é uma ferramenta de linha de comando para gerenciar locks distribuídos
via Redis com suporte a fencing tokens e renovação automática via heartbeat.`,
	}

	root.PersistentFlags().StringVar(&redisAddr, "redis", "localhost:6379", "endereço do servidor Redis")

	root.AddCommand(newAcquireCmd())
	root.AddCommand(newReleaseCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newWatchCmd())

	return root
}
