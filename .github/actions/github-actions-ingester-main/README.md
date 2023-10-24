# Dynatrace Workflow Ingester

This is a GitHub action for ingesting the information about a completed GitHub Actions Workflow
as a Business Event into [Dynatrace Grail](https://www.dynatrace.com/monitoring/platform/dynatrace-grail/). It's
configured for the repository itself.
See [dynatrace-ingest.yml](https://github.com/SemanticlabsGmbH/dynatrace-workflow-ingester/blob/main/.github/workflows/dynatrace-ingest.yml).

Here's an example configuration:

```yaml
name: 'dynatrace-ingest'
on:
  workflow_run:
    workflows: [ my-ci-workflow ] # The Workflow to be ingested.
    types:
      - completed

jobs:
  ingest:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: ./ # to be replaced with public GitHub Action
        with:
          dt-client-id: ${{ secrets.DT_CLIENT_ID }} # Client ID of Dynatrace OAuth Client
          dt-client-secret: ${{ secrets.DT_CLIENT_SECRET }} # Client secret of Dynatrace OAuth Client
          dt-environment-id: ${{ vars.DT_ENVIRONMENT_ID }} # Dynatrace Environment ID
```

This project used https://github.com/actions/typescript-action as a template.
