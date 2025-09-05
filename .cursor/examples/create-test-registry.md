# Local test registry

For testing the features in this folder, you can run a local OCI distribution registry

## Run local registry for tests

To have a local OCI registry to test with, run the registry's docker container locally with the default configuration⁠:

```
docker run -d -p 6000:5000 --restart always --name artifact-cli-test-registry distribution/distribution:edge
```

NOTE: in order to run push/pull against the locally run registry you must allow your docker (containerd) engine to use insecure registry by editing /etc/docker/daemon.json and subsequently restarting it. Although local registries, whose IP address falls in the 127.0.0.0/8 range, are automatically marked as insecure as of Docker 1.3.2. It isn't recommended to rely on this, as it may change in the future.

```
{
     "insecure-registries": ["host.docker.internal:6000"]
}
```

## Delete local registry for tests

Once you're done, if you want to delete the registry, run:

```
docker stop artifact-cli-test-registry
docker rm -vf artifact-cli-test-registry
```
