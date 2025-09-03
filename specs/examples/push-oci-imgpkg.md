# Push an artifact as OCI imgpkg (no platform provided)

## Push
```
./bin/artifact-cli push localhost:5001/educates/my-app-oci-imgpkg:1.0.0 -f ./test/my-app --as imgpkg
```

## Verify

```
crane manifest localhost:5001/educates/my-app-oci-imgpkg:1.0.0 | jq
```

## Result

```
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
      "digest": "sha256:9096f49e98069de10f73ebbdb1fe8e2950d85f32d6dbc6a239af2163e09e4933",
      "size": 243
    }
  ],
  "annotations": {
    "dev.educates.artifact-cli.artifact-type": "imgpkg",
    "dev.educates.artifact-cli.tool": "artifact-cli",
    "dev.educates.artifact-cli.version": "1.0.0",
    "org.opencontainers.image.description": "Folder artifact created by artifact-cli",
    "org.opencontainers.image.title": "artifact-cli artifact"
  }
}
```