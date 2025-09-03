# Tools support compatibility

- `artifact-cli` needs to support pulling oci images generated wih `imgpkg` 
- `artifact-cli` needs to support pushing and pulling images compatible with `docker buildx` format.
- On `artifact-cli push`, if no platform is provided, the generated image will have no platform selector
- On `artifact-cli pull`, if no platform selector is provided, the cli will try to pull an image without 
  platform selection following this criteria:
  - Try to pull an image generated with `artifact-cli push`
  - Try to pull an image generated via `imgpkg`
  - Try to pull an image generated with `docker buildx` using the current architecture as platform architecture
- `artifact-cli` will add annotations to the generated artifact `dev.educates.artifact-cli.tool`  



## Carvel's Imgpkg

- Carvel `imgpkg` github repository can be found at https://github.com/carvel-dev/imgpkg
- `imgpkg` will not generate oci artifacts with index
- `imgpkg` will use mediaType `application/vnd.docker.distribution.manifest.v2+json` for the oci artifact manifest
- `imgpkg` will use mediaType `application/vnd.docker.container.image.v1+json` for the oci config
- `imgpkg` will use mediaType `application/vnd.docker.image.rootfs.diff.tar.gzip` for the oci layers

This is an example manifest of an oci artifact pushed by `imgpkg`
```
{
        "schemaVersion": 2,
        "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
        "config": {
                "mediaType": "application/vnd.docker.container.image.v1+json",
                "size": 273,
                "digest": "sha256:0998a90be3fd120d67a826dd373a7b91837db5ede267da5a99a202eebcf54d4a"
        },
        "layers": [
                {
                        "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
                        "size": 45968059,
                        "digest": "sha256:4f81d41fdf33b1da70704c1050ae815e238fc09c1ee1688924749491314088b8"
                }
        ]
}
```

## Docker Buildx

- `docker buildx` github repository can be found at https://github.com/moby/moby
- `docker buildx` **will** generate oci artifacts with index with mediaType `application/vnd.oci.image.index.v1+json`
- `docker buildx` will generate for every architecture provided a manifest with mediaType `application/vnd.oci.image.manifest.v1+json` and indicating the `platform`

An example of a Docker multiarchitecture oci artifact manifest is:
```
{
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.index.v1+json",
        "manifests": [
           {
              "mediaType": "application/vnd.oci.image.manifest.v1+json",
              "size": 669,
              "digest": "sha256:a0d23f91e20053c1a39ec87f97c63ab55244bab7af69228fa6eb7d75f9cbb00c",
              "platform": {
                 "architecture": "arm64",
                 "os": "linux"
              }
           },
           {
              "mediaType": "application/vnd.oci.image.manifest.v1+json",
              "size": 566,
              "digest": "sha256:8045042db16e4a6f07313e8f68e2e0cac8fe04f79c5ee3027797fd3204af9030",
              "platform": {
                 "architecture": "unknown",
                 "os": "unknown"
              }
           }
        ]
     }
```

- `docker buildx` will use mediaType `application/vnd.docker.distribution.manifest.v2+json` for the oci artifact manifest
- `docker buildx` will use mediaType `application/vnd.docker.container.image.v1+json` for the oci config
- `docker buildx` will use mediaType `application/vnd.docker.image.rootfs.diff.tar.gzip` for the oci layers

An example of an oci image for `linux/arm64` architecture generated via docker buildx is:
```
{
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.manifest.v1+json",
        "config": {
                "mediaType": "application/vnd.oci.image.config.v1+json",
                "digest": "sha256:2578a5e4565ac62de23a161a19040d6ff04a8311bb274cea5d4cac261dcbed5e",
                "size": 804
        },
        "layers": [
                {
                        "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
                        "digest": "sha256:b99300be2662a4e896c1c6546adfd193378459bebd39ff0420ad923a46fc811a",
                        "size": 278
                },
                {
                        "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
                        "digest": "sha256:1046ddf8815bd91c26c2d3180aeb28488fd71f0b302bc29f07c01c4e74c82c96",
                        "size": 42765975
                }
        ]
}
```