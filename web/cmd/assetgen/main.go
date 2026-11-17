// Command assetgen writes the public asset manifest before Go compilation.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nzinovev/nize-echo/internal/staticassets"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	manifest, err := staticassets.Build(os.DirFS("static"))
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode static asset manifest: %w", err)
	}
	data = append(data, '\n')
	const output = "static-assets.json"
	current, err := os.ReadFile(output)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read static asset manifest: %w", err)
	}
	if bytes.Equal(data, current) {
		return nil
	}
	tmp, err := os.CreateTemp(".", ".static-assets-*")
	if err != nil {
		return fmt.Errorf("create temporary asset manifest: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil {
		return fmt.Errorf("write static asset manifest: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close static asset manifest: %w", closeErr)
	}
	if err := os.Rename(tmp.Name(), output); err != nil {
		return fmt.Errorf("write static asset manifest: %w", err)
	}
	return nil
}
