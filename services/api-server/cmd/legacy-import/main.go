// 旧数据导入命令行入口

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/legacyimport"
)

// 程序入口
func main() {
	mode := flag.String("mode", "preflight", "preflight or import")
	source := flag.String("source", "", "path to legacy SQLite database (opened read-only)")
	target := flag.String("target", "", "new Go SQLite database path; required for import")
	allowPartial := flag.Bool("allow-partial", false, "explicitly permit skipped rows or sidecar files")
	recordingsDir := flag.String("recordings-dir", "", "new directory for legacy recording sidecars")
	flag.Parse()
	if *source == "" || (*mode == "import" && *target == "") {
		fmt.Fprintln(os.Stderr, "usage: legacy-import -source old.db [-mode preflight|import -target new.db]")
		os.Exit(2)
	}
	var report legacyimport.Report
	var err error
	switch *mode {
	case "preflight":
		report, err = legacyimport.Preflight(context.Background(), *source)
	case "import":
		report, err = legacyimport.ImportWithOptions(context.Background(), *source, *target, legacyimport.Options{AllowPartial: *allowPartial, RecordingsDir: *recordingsDir})
	default:
		fmt.Fprintln(os.Stderr, "invalid mode:", *mode)
		os.Exit(2)
	}
	if len(report.Tables) > 0 {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(report); encodeErr != nil {
			fmt.Fprintln(os.Stderr, encodeErr)
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
