package functiondockerfile

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

const (
	jvmRuntimeBaseImage = "adoptopenjdk/openjdk11:x86_64-alpine-jre-11.0.10_9"
	mavenOpenJDKImage   = "maven:3-openjdk-11"
)

func javaGenerator(f *stack.Function, version, provider string, w io.Writer) error {
	buildCon, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   mavenOpenJDKImage,
		As:     "build",
		Ignore: []string{},
	})
	if err != nil {
		return err
	}
	err = mavenBuild(buildCon, f)
	if err != nil {
		return err
	}

	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   jvmRuntimeBaseImage,
		Ignore: []string{},
	})
	if err != nil {
		return err
	}

	con.Copy(dockerfile.CopyOptions{Src: f.Handler, Dest: "function.jar", From: "build"})
	con.Config(dockerfile.ConfigOptions{
		WorkingDir: "/",
		Ports:      []int32{9001},
		Cmd:        []string{"java", "-jar", "function.jar"},
	})
	withMembrane(con, version, provider)

	_, err = w.Write([]byte(strings.Join(append(buildCon.Lines(), con.Lines()...), "\n")))
	return err
}

func glob(dir string, name string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.Name() == name {
			// remove the provided dir (so it's like we have changed dir here)
			files = append(files, strings.Replace(path, dir, "", 1))
		}
		return nil
	})

	return files, err
}

func mavenBuild(con dockerfile.ContainerState, f *stack.Function) error {
	pomFiles, err := glob(f.ContextDirectory(), "pom.xml")
	if err != nil {
		return err
	}

	moduleDirs := []string{}
	for _, p := range pomFiles {
		moduleDirs = append(moduleDirs, path.Dir(p))
		con.Copy(dockerfile.CopyOptions{Src: p, Dest: path.Join("./", p)})
	}
	con.Run(dockerfile.RunOptions{Command: []string{"mvn", "de.qaware.maven:go-offline-maven-plugin:resolve-dependencies"}})
	for _, d := range moduleDirs {
		con.Copy(dockerfile.CopyOptions{Src: d, Dest: path.Join("./", d)})
	}
	con.Run(dockerfile.RunOptions{Command: []string{"mvn", "clean", "package"}})
	return nil
}
