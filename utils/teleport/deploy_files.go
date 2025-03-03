package teleport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/docker/compose/v2/pkg/progress"
	"github.com/sirupsen/logrus"
	"github.com/varrcan/dl/project"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type callMethod struct{}

func copyFiles(ctx context.Context, t *teleport, override []string) {
	var (
		err  error
		path string
	)

	w := progress.ContextWriter(ctx)
	w.Event(progress.Event{ID: "Files", Status: progress.Working})

	path = "bitrix"
	if len(override) > 0 {
		path = strings.Join(override, " ")
	}

	logrus.Infof("Download path from server: %s", path)
	err = t.packFiles(ctx, path)
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		os.Exit(1)
	}

	err = t.downloadArchive(ctx)
	if err != nil {
		w.Event(progress.Event{ID: "Files", Status: progress.Error})
		return
	}

	err = project.ExtractArchive(ctx, path)
	if err != nil {
		w.Event(progress.Event{ID: "Files", Status: progress.Error})
		return
	}

	var a project.CallMethod
	reflect.
		ValueOf(&a).
		MethodByName(cases.Title(language.Und, cases.NoLower).String("BitrixAccess")).
		Call([]reflect.Value{})

	w.Event(progress.Event{ID: "Files", Status: progress.Done})
}

func (t *teleport) packFiles(ctx context.Context, path string) error {
	w := progress.ContextWriter(ctx)

	w.Event(progress.Event{ID: "Archive files", ParentID: "Files", Status: progress.Working})

	excludeTarString := project.FormatIgnoredPath()
	tarCmd := strings.Join([]string{"cd", t.Catalog, "&&",
		"tar",
		"--dereference",
		"-zcf",
		"production.tar.gz",
		excludeTarString,
		path,
	}, " ")
	logrus.Infof("Run archiving files: %s", tarCmd)
	_, err := t.run(tarCmd)

	if err != nil {
		return err
	}

	w.Event(progress.Event{ID: "Archive files", ParentID: "Files", Status: progress.Done})

	return nil
}

func (t *teleport) downloadArchive(ctx context.Context) error {
	w := progress.ContextWriter(ctx)

	serverPath := filepath.Join(t.Catalog, "production.tar.gz")
	localPath := filepath.Join(project.Env.GetString("PWD"), "production.tar.gz")

	w.Event(progress.Event{ID: "Download archive", ParentID: "Files", Status: progress.Working})

	logrus.Infof("Download archive: %s", serverPath)
	err := t.download(serverPath, localPath)

	if err != nil {
		w.Event(progress.ErrorMessageEvent("Download error", fmt.Sprint(err)))
		return err
	}

	logrus.Infof("Delete archive: %s", serverPath)
	err = t.delete(serverPath)
	if err != nil {
		w.Event(progress.ErrorMessageEvent("File deletion error", fmt.Sprint(err)))
		return err
	}

	w.Event(progress.Event{ID: "Download archive", ParentID: "Files", Status: progress.Done})

	return err
}
