# OrderPulse

GitOps-driven progressive delivery for an order-processing service: canary
releases gated on live Prometheus metrics, with fully automated rollback on
regression, and a designed (not live-wired) AI-assisted incident-triage step.

## What this actually demonstrates

Most portfolio Kubernetes projects stop at "deployed an app to a cluster."
This one is about what happens next: shipping a change safely, watching it
prove itself against real traffic before it gets full rollout, and backing
off automatically the moment it doesn't. That is the core of what
"progressive delivery" and "canary analysis" mean in a real production
context, and this repo is a working, tested implementation of it, not a
diagram.

## Architecture

    Azure VM (Standard_D2as_v5, single node)
    Terraform provisions: resource group, network, NSG (SSH-only), VM
    cloud-init bootstraps k3s on first boot

    k3s cluster
      Argo CD          watches gitops/, auto-syncs, self-heals, prunes
      Argo Rollouts     replaces plain Deployments with staged canary releases
      kube-prometheus-stack   Prometheus + Grafana + Alertmanager
      orders-api        Go service, AI-generated, Prometheus-instrumented
      load-gen          CronJob generating continuous synthetic traffic

    GitHub Actions CI
      push to main -> lint/vet -> build -> push image to GHCR
                    -> bump image tag in gitops/rollout.yaml -> commit -> push
    Argo CD detects the change and starts a canary rollout automatically.

## How the canary analysis works

`gitops/rollout.yaml` defines a staged rollout: 20% traffic, pause, run
analysis; 50% traffic, pause, run analysis; 100%. Each analysis step queries
Prometheus directly (`gitops/analysistemplate.yaml`) for two live signals:

- error rate on `/orders` (5xx responses / total requests)
- p95 latency on `/orders`

If either metric breaches its threshold across the sampled window, Argo
Rollouts aborts the update automatically, scales the canary to zero, and
reverts all traffic to the last known-good revision. No human runs
`kubectl rollout undo` — the decision is made from data, not a person
watching a dashboard.

## Proven, not just designed

This was tested live against the running cluster, not just written:

- **Automated abort**: a build with an injected failure rate (`FAIL_RATE`
  env var in `orders-api`) was deployed. The canary reached 20% weight, the
  p95-latency analysis failed 2 of 3 sampled windows, and Argo Rollouts
  aborted automatically — full revert to stable, zero manual intervention.
- **Clean promotion**: the same fix, reverted, produced a canary that passed
  both analysis gates and promoted cleanly to 100%.

Both runs are visible in the Argo Rollouts revision history on the live
cluster; see `docs/example-incident-summary.md` for the actual abort
message and an example of the AI-generated incident summary that would be
filed automatically in a live-billed version of this pipeline.

## Monitoring

- `gitops/servicemonitor.yaml` — Prometheus scrape config for orders-api
- `gitops/prometheusrule.yaml` — 3 alert rules: high error rate, high
  latency, and target-down (a distinct failure mode a latency/error alert
  alone would miss entirely)
- `monitoring/grafana-dashboard.json` — request rate by status, error rate
  %, and p95 latency broken out by pod (so a canary pod's degradation is
  visible separately from stable pods during a rollout)

## AI-assisted development

- `orders-api`'s application code was AI-generated end to end; engineering
  effort on this project went entirely into the DevOps/infra/delivery
  layer, not the app logic.
- `scripts/rollout-watcher.py` is a designed-and-committed (not live-wired)
  incident-triage automation: on a real abort, it would gather the Rollout
  status message, AnalysisRun results, and recent pod logs, send them to
  the Claude API, and file the returned summary as a GitHub Issue
  automatically. Left unwired in this build deliberately, to avoid a paid
  API dependency for a time-boxed portfolio project — see
  `docs/example-incident-summary.md` for a representative example of its
  output against this project's real abort data.

## Repo structure

    orderpulse/
    ├── app/                  orders-api (Go), Dockerfile
    ├── gitops/               Argo CD Application, Rollout, Service,
    │                         AnalysisTemplate, ServiceMonitor, PrometheusRule,
    │                         load-gen CronJob
    ├── infra/                Terraform: VM, network, NSG, cloud-init
    ├── monitoring/            kube-prometheus-stack values, Grafana dashboard
    ├── scripts/               AI incident-triage watcher (design, unwired)
    ├── docs/                  example incident summary
    └── .github/workflows/     CI: build, vet, push to GHCR, bump gitops tag

## Reproducing the rollback locally

1. Edit `gitops/rollout.yaml`, add `FAIL_RATE: "0.6"` as an env var on the
   `orders-api` container.
2. Commit and push. Argo CD picks it up automatically.
3. Watch: `kubectl argo rollouts get rollout orders-api --watch`
4. Observe the canary abort at the analysis gate and full automatic revert
   to stable.
5. Remove the env var, commit, push — watch a clean promotion instead.

## What I'd add with more time

- Split `app/` and `gitops/` into separate repos with a real deploy-key
  boundary, closer to actual app-team/platform-team separation.
- A second service, so the canary story includes a cross-service failure
  mode, not just a single service's own regression.
- Remote-write Prometheus storage so metrics history survives VM teardown.
- SLO-based error budgets/burn-rate alerts instead of static thresholds.
- Wire `scripts/rollout-watcher.py` to a live, billed API key and run it as
  an in-cluster CronJob rather than host cron.
- Bring back a DevSecOps layer (image scanning, signing) as a deliberate
  follow-on, once the delivery mechanics were already proven solid.

## Notes on constraints

Built in a hard time/cost budget: single Azure VM, no AKS, no service mesh,
no DevSecOps tooling — all deliberate scope decisions to keep effort
focused on progressive delivery and observability, the actual point of the
project, documented above rather than treated as unexplained omissions.
