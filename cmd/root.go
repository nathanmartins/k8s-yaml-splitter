package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "yaml-split",
	Short: "this command will transform a list of yaml documents into individual documents",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(_ *cobra.Command, args []string) {

		var firstFileBytes []byte
		var err error

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		if len(args) < 2 { //nolint:mnd // not magic number
			firstFileBytes, err = io.ReadAll(os.Stdin)
		} else {
			fName := args[0]
			firstFileBytes, err = ReadFileBytes(fName)
		}

		if err != nil {
			logger.Error("error reading file", "err", err)
			os.Exit(-1)
		}

		firstFile, err := kio.FromBytes(firstFileBytes)
		if err != nil {
			logger.Error("error reading file", "err", err)
			os.Exit(-1)
		}

		result := args[0]

		if len(args) > 1 {
			result = args[1]
		}

		err = os.MkdirAll(result, 0750)
		if err != nil {
			logger.Error("error creating directory", "err", err)
			os.Exit(-1)
		}

		for _, node := range firstFile {

			fileName := fmt.Sprintf("%s-%s.yaml", strings.ToLower(node.GetKind()), strings.ToLower(node.GetName()))
			fileName = strings.ReplaceAll(fileName, ":", "-")
			logger.Info("processing", "fileName", fileName)

			OverWriteToFile(
				fmt.Sprintf("%s/%s", result, fileName),
				node.MustString(),
				logger,
			)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.k8s-yaml-splitter.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// ReadFileBytes will read the entire contents of a file and return it as an array of bytes.
func ReadFileBytes(filePath string) ([]byte, error) {
	b, err := os.ReadFile(filePath) // just pass the file name
	return b, err
}

// OverWriteToFile this will truncate the file and write over any contents.
func OverWriteToFile(filePath string, payload string, logger *slog.Logger) {
	var f *os.File

	// If the file doesn't exist, create it, or append to the file.
	f, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0600,
	)
	if err != nil {
		logger.Error("error opening file", "err", err)
		os.Exit(-1)
	}
	if _, err = f.WriteString(payload); err != nil {
		logger.Error("error writing to file", "err", err)
		os.Exit(-1)
	}
	if err = f.Close(); err != nil {
		logger.Error("error closing file", "err", err)
		os.Exit(-1)
	}
}
