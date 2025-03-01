package command

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/varrcan/dl/project"
	"github.com/varrcan/dl/utils/docker"
	"golang.org/x/sync/errgroup"
)

type containerSummary struct {
	ID         string
	Name       string
	State      string
	Health     string
	IPAddress  string
	ExitCode   int
	Publishers portPublishers
}

type portPublishers []portPublisher

type portPublisher struct {
	URL           string
	TargetPort    int
	PublishedPort int
	Protocol      string
}

func psCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		Long:  `List containers in the current project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runPs()
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func runPs() error {
	project.LoadEnv()
	ctx := context.Background()

	cli, err := docker.NewClient()
	if err != nil {
		pterm.Fatal.Printfln("Failed to connect to socket")
		return err
	}

	networkName := project.Env.GetString("NETWORK_NAME")
	containers, err := getProjectContainers(ctx, cli, networkName)

	var data [][]string
	data = append(data, []string{"ID", "Name", "State", "IP", "Ports"})
	for _, container := range containers {
		status := container.State
		if status == "running" && container.Health != "" {
			status = fmt.Sprintf("%s (%s)", container.State, container.Health)
		} else if status == "exited" || status == "dead" {
			status = fmt.Sprintf("%s (%d)", container.State, container.ExitCode)
		}
		con := []string{container.ID[:12], container.Name, status, container.IPAddress, displayablePorts(container)}
		data = append(data, con)
	}

	err = pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	if err != nil {
		return err
	}

	return err
}

func getProjectContainers(ctx context.Context, cli *docker.Client, projectName string) ([]containerSummary, error) {
	containerFilter := filters.NewArgs(filters.Arg("label", fmt.Sprintf("%s=%s", api.ProjectLabel, projectName)))
	containers, _ := cli.ContainerList(ctx, types.ContainerListOptions{Filters: containerFilter, All: true})

	netFilters := filters.NewArgs(filters.Arg("name", projectName+"_default"))
	network, _ := cli.NetworkList(ctx, types.NetworkListOptions{Filters: netFilters})

	summary := make([]containerSummary, len(containers))
	eg, ctx := errgroup.WithContext(ctx)
	for i, container := range containers {
		i, container := i, container
		eg.Go(func() error {
			var publishers []portPublisher
			sort.Slice(container.Ports, func(i, j int) bool {
				return container.Ports[i].PrivatePort < container.Ports[j].PrivatePort
			})
			for _, p := range container.Ports {
				publishers = append(publishers, portPublisher{
					URL:           p.IP,
					TargetPort:    int(p.PrivatePort),
					PublishedPort: int(p.PublicPort),
					Protocol:      p.Type,
				})
			}

			inspect, err := cli.ContainerInspect(ctx, container.ID)
			if err != nil {
				return err
			}

			var (
				ip       string
				health   string
				exitCode int
			)
			if inspect.State != nil {
				switch inspect.State.Status {
				case "running":
					if inspect.State.Health != nil {
						health = inspect.State.Health.Status
					}
				case "exited", "dead":
					exitCode = inspect.State.ExitCode
				}
			}

			for _, n := range container.NetworkSettings.Networks {
				if network[0].ID == n.NetworkID {
					ip = n.IPAddress
				}
			}

			summary[i] = containerSummary{
				ID:         container.ID,
				Name:       docker.GetCanonicalContainerName(container),
				State:      container.State,
				Health:     health,
				ExitCode:   exitCode,
				Publishers: publishers,
				IPAddress:  ip,
			}
			return nil
		})
	}
	return summary, eg.Wait()
}

func displayablePorts(c containerSummary) string {
	if c.Publishers == nil {
		return ""
	}

	ports := make([]types.Port, len(c.Publishers))
	for i, pub := range c.Publishers {
		ports[i] = types.Port{
			IP:          pub.URL,
			PrivatePort: uint16(pub.TargetPort),
			PublicPort:  uint16(pub.PublishedPort),
			Type:        pub.Protocol,
		}
	}

	return formatter.DisplayablePorts(ports)
}
