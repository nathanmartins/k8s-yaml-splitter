package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"strings"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-yaml-splitter",
	Short: "this command will transform a list of yaml documents into individual documents",
	Args:  cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("YamlSplitCmd")

		fName := args[0]

		firstFileBytes, err := ReadFileBytes(fName)
		if err != nil {
			log.Fatalln(err)
		}

		firstFile, err := kio.FromBytes(firstFileBytes)
		if err != nil {
			log.Fatalln(err)
		}

		err = os.MkdirAll(args[1], os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		for _, node := range firstFile {
			OverWriteToFile(fmt.Sprintf("%s/%s-%s.yaml", args[1], strings.ToLower(node.GetKind()), strings.ToLower(node.GetName())), node.MustString())
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

// ReadFileBytes will read the entire contents of a file and return it as an array of bytes
func ReadFileBytes(filePath string) ([]byte, error) {
	b, err := os.ReadFile(filePath) // just pass the file name
	return b, err
}

// OverWriteToFile this will truncate the file and write over any contents
func OverWriteToFile(filePath string, payload string) {
	var f *os.File

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = f.Write([]byte(payload)); err != nil {
		log.Fatal(err)
	}
	if err = f.Close(); err != nil {
		log.Fatal(err)
	}
}
