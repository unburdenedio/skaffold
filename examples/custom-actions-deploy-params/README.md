# Custom actions: deploy parameters

This example demonstrates how `skaffold exec` forwards deploy parameters
— values supplied on the command line with `--set` or `--set-value-file`
— as environment variables into every container of the invoked custom
action. This mirrors [Google Cloud Deploy custom
targets](https://cloud.google.com/deploy/docs/custom-targets#custom_targets_and_deploy_parameters).

## Try it

```console
$ skaffold exec show-params \
    --set TF_VAR_bucket=my-bkt \
    --set REGION=us-central1
```

The `printer` container will echo:

```
TF_VAR_bucket=my-bkt
REGION=us-central1
```

## Precedence

When the same key appears in multiple sources, later entries win:

1. `--verify-env-file` / base env
2. `--set-value-file`
3. `--set` (highest priority)

On key collision, deploy parameters override a container's own `env:`
entry declared in `skaffold.yaml`.
