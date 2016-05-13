package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"net/http"

	apiReq "github.com/AcalephStorage/kontinuous/cli/request"
	"github.com/codegangsta/cli"
	"github.com/gosuri/uitable"
)

func main() {
	app := cli.NewApp()

	app.Name = "kontinuous-cli"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "conf, c",
			Value: "./config",
			Usage: "Specify an alternate configuratioxn file (default: ./config)",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "get",
			Subcommands: []cli.Command{
				{
					Name:      "pipelines",
					Usage:     "get all pipelines",
					ArgsUsage: "[pipeline-name]",
					Before: func(c *cli.Context) error {
						p := strings.TrimSpace(c.Args().First())
						if len(p) > 0 {
							return requireNameArg(c)
						}
						return nil
					},
					Action: getPipelines,
				},
				{
					Name:   "repos",
					Usage:  "get all repositories",
					Action: getRepos,
				},
				{
					Name:      "builds",
					Usage:     "get all builds of pipeline",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    getBuilds,
				},
				{
					Name:      "stages",
					Usage:     "get the stages of a pipeline build",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "build, b",
							Usage: "build number, if not provided will get stages of latest build",
						},
					},
					Action: getStages,
				},
			},
		},
		{
			Name: "create",
			Subcommands: []cli.Command{
				{
					Name:      "pipeline",
					Usage:     "create pipeline for repo",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "events",
							Value: "push",
						},
					},
					Action: createPipeline,
				},
				{
					Name:      "build",
					Usage:     "trigger pipeline build",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    createBuild,
				},
			},
		},
		{
			Name: "delete",
			Subcommands: []cli.Command{
				{
					Name:      "pipeline",
					Usage:     "delete pipeline",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    deletePipeline,
				},
				{
					Name:      "build",
					Usage:     "delete build",
					ArgsUsage: "<pipeline-name>",
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "build, b",
							Usage: "build number, if not provided will not proceed deletion",
						},
					},
					Before: requireNameArg,
					Action: deleteBuild,
				},
			},
		},
		{
			Name:  "deploy",
			Usage: "deploy kontinuous app in the cluster",
			Subcommands: []cli.Command{
				{
					Name:   "remove",
					Usage:  "remove kontinuous app in the cluster",
					Action: removeDeployedApp,
				},
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "namespace",
					Usage: "Required, kontinuous namespace",
				},
				cli.StringFlag{
					Name:  "accesskey",
					Usage: "Required, s3 access key",
				},
				cli.StringFlag{
					Name:  "secretkey",
					Usage: "Required, s3 secret key",
				},
				cli.StringFlag{
					Name:  "authcode",
					Usage: "Required, jwt authorization code",
				},
			},

			Action: deployApp,
		},
	}
	app.Run(os.Args)
}

func requireNameArg(c *cli.Context) error {
	if _, _, err := parseNameArg(c.Args().First()); err != nil {
		return err
	}
	return nil
}

func parseNameArg(name string) (owner, repo string, err error) {
	invalid := errors.New("Invalid pipeline name")
	required := errors.New("Provide pipeline name")

	p := strings.TrimSpace(name)
	if len(p) == 0 {
		return "", "", required
	}
	fullName := strings.Split(p, "/")
	if len(fullName) != 2 {
		return "", "", invalid
	}

	owner = fullName[0]
	repo = fullName[1]
	if len(owner) == 0 || len(repo) == 0 {
		return "", "", invalid
	}
	return owner, repo, nil
}

// ACTIONS

func getPipelines(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	pipelineName := c.Args().First()
	pipelines, err := config.GetPipelines(http.DefaultClient, pipelineName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("NAME")
	for _, p := range pipelines {
		name := fmt.Sprintf("%s/%s", p.Owner, p.Repo)
		table.AddRow(name)
	}
	fmt.Println(table)
}

func getRepos(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	repos, err := config.GetRepos(http.DefaultClient)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("OWNER", "NAME")
	for _, r := range repos {
		table.AddRow(r.Owner, r.Name)
	}
	fmt.Println(table)
}

func getBuilds(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}

	owner, repo, _ := parseNameArg(c.Args().First())
	builds, err := config.GetBuilds(http.DefaultClient, owner, repo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("BUILD", "STATUS", "CREATED", "FINISHED", "EVENT", "AUTHOR", "COMMIT")
	for _, b := range builds {
		created := "-"
		if b.Created != 0 {
			created = time.Unix(0, b.Created).Format(time.RFC3339)
		}
		finished := "-"
		if b.Finished != 0 {
			finished = time.Unix(0, b.Finished).Format(time.RFC3339)
		}
		table.AddRow(b.Number, b.Status, created, finished, b.Event, b.Author, b.Commit)
	}
	fmt.Println(table)
}

func getStages(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	stages, err := config.GetStages(http.DefaultClient, owner, repo, c.Int("build"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("INDEX", "TYPE", "NAME", "STATUS", "STARTED", "FINISHED")
	for _, s := range stages {
		started := "-"
		if s.Started != 0 {
			started = time.Unix(0, s.Started).Format(time.RFC3339)
		}
		finished := "-"
		if s.Finished != 0 {
			finished = time.Unix(0, s.Finished).Format(time.RFC3339)
		}
		table.AddRow(s.Index, s.Type, s.Name, s.Status, started, finished)
	}
	fmt.Println(table)
}

func createPipeline(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	events := strings.Split(c.String("events"), ",")
	for i, e := range events {
		events[i] = strings.TrimSpace(e)
	}

	pipeline := &apiReq.PipelineData{
		Owner:  owner,
		Repo:   repo,
		Events: events,
	}

	err = config.CreatePipeline(http.DefaultClient, pipeline)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("pipeline `%s/%s` created", pipeline.Owner, pipeline.Repo)
	}
}

func createBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	err = config.CreateBuild(http.DefaultClient, owner, repo)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("building pipeline %s/%s ", owner, repo)
	}
}

func deletePipeline(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	pipelineName := c.Args().First()
	err = config.DeletePipeline(http.DefaultClient, pipelineName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("pipeline %s successfully deleted.\n", pipelineName)
}

func deleteBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}

	pipelineName := c.Args().First()
	buildNum := c.String("build")
	err = config.DeleteBuild(http.DefaultClient, pipelineName, buildNum)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("pipeline %s build #%s successfully deleted.\n", pipelineName, buildNum)
}

func deployApp(c *cli.Context) {
	namespace := c.String("namespace")
	accessKey := c.String("accesskey")
	secretKey := c.String("secretkey")
	authCode := c.String("authcode")

	missingFields := false
	if namespace == "" || accessKey == "" || secretKey == "" || authCode == "" {
		fmt.Println("missing required fields!")
		missingFields = true

	}

	if !missingFields {
		err := DeployKontinuous(namespace, accessKey, secretKey, authCode)
		if err != nil {
			fmt.Println("Oops something went wrong. Unable to deploy kontinuous.")
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Success! Kontinuous is now deployed in the cluster.")
	}
}

func removeDeployedApp(c *cli.Context) {
	err := RemoveResources()

	if err != nil {
		fmt.Println("Oops something went wrong. Unable to remove kontinuous.")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Success! Kontinuous app has been removed from the cluster. ")
}
