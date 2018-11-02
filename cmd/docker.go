package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stellar/go/keypair"
)

func findContainersByPrefix(cli *client.Client, p string) (containers []types.Container, err error) {
	ctx := context.Background()
	var cl []types.Container
	if cl, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}

	for _, i := range cl {
		for _, name := range i.Names {
			if !strings.HasPrefix(name[1:], "scn.") {
				continue
			}

			containers = append(containers, i)
			break
		}
	}

	return
}

func findContainer(cli *client.Client, s string) (c types.Container, err error) {
	ctx := context.Background()
	var cl []types.Container
	if cl, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}

	for _, i := range cl {
		for _, name := range i.Names {
			if !strings.Contains(name[1:], s) {
				continue
			}

			c = i
			return
		}
	}

	return
}

func removeContainerByName(cli *client.Client, s string) (err error) {
	var info types.Container
	if info, err = findContainer(cli, s); err != nil {
		return
	}

	if len(info.ID) < 1 {
		return
	}

	return removeContainerByID(cli, info.ID)
}

func removeContainerByID(cli *client.Client, s string) (err error) {
	err = cli.ContainerRemove(context.Background(), s, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		log.Error("failed to remove container", "error", err, "container", s)
		return
	}
	log.Debug("container removed", "container", s)

	return
}

func cleanDocker(cli *client.Client) (err error) {
	ctx := context.Background()
	var cl []types.Container
	if cl, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}

	for _, c := range cl {
		log.Debug("found container", "container", c)
		for _, name := range c.Names {
			if !strings.HasPrefix(name[1:], dockerContainerNamePrefix) {
				continue
			}

			// remove container :)
			if err = cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
				log.Error("failed to remove container", "error", err, "container", c)
				return
			}
			log.Debug("container removed", "container", c)
		}
	}

	return
}

func findImage(cli *client.Client, s string) (id string, err error) {
	ctx := context.Background()

	var images []types.ImageSummary
	if images, err = cli.ImageList(ctx, types.ImageListOptions{All: true}); err != nil {
		log.Error("failed to get image list", "error", err)
		return
	}

	for _, i := range images {
		for _, t := range i.RepoTags {
			if strings.Contains(t, s) {
				id = i.ID
				return
			}
		}
	}

	return
}

func removeImage(cli *client.Client, s string) (err error) {
	var id string
	if id, err = findImage(cli, s); err != nil {
		return
	}

	if len(id) < 1 {
		err = fmt.Errorf("image not found: '%s'", s)
		return
	}

	var resp []types.ImageDelete
	resp, err = cli.ImageRemove(context.Background(), id, types.ImageRemoveOptions{})
	if err != nil {
		return
	}
	for _ = range resp {
	}

	return
}

func pullImage(cli *client.Client, t string) (err error) {
	var resp io.ReadCloser
	resp, err = cli.ImagePull(
		context.Background(),
		t,
		types.ImagePullOptions{},
	)
	if err != nil {
		return
	}

	_, _ = ioutil.ReadAll(resp)
	return
}

func runContainerGettingIP(cli *client.Client) (ip string, err error) {
	var imageID string
	if imageID, err = findImage(cli, "alpine:latest"); err != nil {
		return
	}

	if len(imageID) < 1 {
		err = pullImage(cli, "docker.io/library/alpine:latest")
		if err != nil {
			return
		}
		if imageID, err = findImage(cli, "alpine:latest"); err != nil {
			return
		} else if len(imageID) < 1 {
			err = fmt.Errorf("failed to pull alpine:latest")
			return
		}
	}

	containerName := "sebak-network-composer-get-ip"

	if err = removeContainerByName(cli, containerName); err != nil {
		return
	}

	containerConfig := &container.Config{
		Image:        imageID,
		AttachStdin:  false,
		AttachStdout: false,
		Tty:          false,
		OpenStdin:    false,
		Entrypoint: []string{
			"/bin/sh",
			"-c",
			"/sbin/ip route | grep '^default ' | sed -e 's/.* src //g' -e 's/ metric.*//g'",
		},
	}
	containerHostConfig := &container.HostConfig{
		NetworkMode: "host",
		//AutoRemove:  true,
	}

	ctx := context.Background()
	var containerBody container.ContainerCreateCreatedBody
	containerBody, err = cli.ContainerCreate(
		ctx,
		containerConfig,
		containerHostConfig,
		&network.NetworkingConfig{},
		containerName,
	)

	if err != nil {
		log.Error("failed to create container", "error", err)
		return
	}

	if err = cli.ContainerStart(ctx, containerBody.ID, types.ContainerStartOptions{}); err != nil {
		log.Error("failed to start container", "error", err)
		return
	}
	defer removeContainerByID(cli, containerBody.ID)

	if _, err = cli.ContainerWait(ctx, containerBody.ID); err != nil {
		return
	}

	out, err := cli.ContainerLogs(ctx, containerBody.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	var b []byte
	if b, err = ioutil.ReadAll(out); err != nil {
		return
	}

	ip = strings.TrimSpace(string(b[8:]))

	return
}

func runSEBAK(dh *DockerHost, nd *node.LocalNode, genesis string, commonKeypair *keypair.Full) (id string, err error) {
	cli := dh.Client()

	ctx := context.Background()

	var images []types.ImageSummary
	if images, err = cli.ImageList(ctx, types.ImageListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}
	if len(images) < 1 {
		err = errors.New("image not found")
		return
	}

	var imageID string
	for _, i := range images {
		if _, found := common.InStringArray(i.RepoTags, flagImageName); !found {
			continue
		}
		imageID = i.ID
		break
	}

	if len(imageID) < 1 {
		err = fmt.Errorf("image not found", flagImageName)
		log.Error("failed to find the image", "image", flagImageName, "error", err)
		return
	}

	var env_validators []string
	for _, v := range nd.GetValidators() {
		s := fmt.Sprintf("%s?address=%s", v.Endpoint(), v.Address())
		env_validators = append(env_validators, s)
	}

	_, port, _ := net.SplitHostPort(nd.Endpoint().Host)
	bindEndpoint, _ := common.NewEndpointFromString(nd.Endpoint().String())
	bindEndpoint.Host = fmt.Sprintf("0.0.0.0:%s", port)

	envs := []string{
		fmt.Sprintf("SEBAK_NODE_ALIAS=%s", nd.Alias()),
		"SEBAK_TLS_CERT=/sebak.crt",
		"SEBAK_TLS_KEY=/sebak.key",
		fmt.Sprintf("SEBAK_LOG_LEVEL=%s", flagSebakLogLevel),
		fmt.Sprintf("SEBAK_SECRET_SEED=%s", nd.Keypair().Seed()),
		fmt.Sprintf("SEBAK_NETWORK_ID=%s", networkID),
		fmt.Sprintf("SEBAK_BIND=%s", bindEndpoint.String()),
		fmt.Sprintf("SEBAK_PUBLISH=%s", nd.Endpoint().String()),
		fmt.Sprintf("SEBAK_GENESIS_BLOCK=%s", genesis),
		fmt.Sprintf("SEBAK_COMMON_ACCOUNT=%s", commonKeypair.Seed()),
		fmt.Sprintf("SEBAK_VALIDATORS=self %s", strings.Join(env_validators, " ")),
	}
	envs = append(envs, dh.Env...)

	var mounts []mount.Mount
	for _, v := range dh.Volume {
		m := mount.Mount{Type: mount.TypeBind, Source: v.Source, Target: v.Target}
		mounts = append(mounts, m)
	}

	containerConfig := &container.Config{
		Image:        imageID,
		AttachStdin:  false,
		AttachStdout: false,
		ExposedPorts: nat.PortSet{nat.Port(port): {}},
		Tty:          false,
		OpenStdin:    false,
		Entrypoint:   []string{"/bin/sh", "/entrypoint.sh"},
		Env:          envs,
	}
	containerHostConfig := &container.HostConfig{
		Mounts: mounts,
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", basePort)): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		NetworkMode: "host",
	}

	var containerBody container.ContainerCreateCreatedBody
	containerBody, err = cli.ContainerCreate(
		ctx,
		containerConfig,
		containerHostConfig,
		&network.NetworkingConfig{},
		makeContainerName(nd),
	)
	if err != nil {
		log.Error("failed to create container", "error", err)
		return
	}

	if err = cli.ContainerStart(ctx, containerBody.ID, types.ContainerStartOptions{}); err != nil {
		log.Error("failed to start container", "error", err)
		return
	}

	id = containerBody.ID

	return
}

func makeContainerName(nd *node.LocalNode) string {
	return fmt.Sprintf("%s%s", dockerContainerNamePrefix, nd.Alias()[:4])
}
