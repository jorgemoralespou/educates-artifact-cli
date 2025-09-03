# Push an artifact as OCI image providing one platform

## Push
```
./bin/artifact-cli push localhost:5001/educates/my-app-oci-linuxarm:1.0.0 -f ./test/my-app -p linux/arm64 --as oci
```

## Verify

```
crane manifest localhost:5001/educates/my-app-oci-linuxarm:1.0.0 | jq
```

## Result

```
crane manifest localhost:5001/educates/my-app-oci-linuxarm:1.0.0 | jq
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:783a9e50c22fd7ae9cf816d9dbe833b82f0dc530b2940eb0f00ecfc58bd4b054",
      "size": 743,
      "platform": {
        "architecture": "arm64",
        "os": "linux"
      }
    }
  ],
  "annotations": {
    "dev.educates.artifact-cli.artifact-type": "oci",
    "dev.educates.artifact-cli.tool": "artifact-cli",
    "dev.educates.artifact-cli.version": "1.0.0",
    "org.opencontainers.image.description": "Folder artifact created by artifact-cli",
    "org.opencontainers.image.platform": "linux/arm64",
    "org.opencontainers.image.title": "artifact-cli artifact"
  }
}
```