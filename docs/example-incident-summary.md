Example AI-generated incident summary

Captured from a real abort during this build (revision 4, canary at 20% weight).

Raw Rollout status message:
RolloutAborted: Rollout aborted update to revision 4: Step-based analysis phase error/failed: Metric "p95-latency" assessed Failed due to failed (2) failureLimit (1)

AI-generated summary (from the design in scripts/rollout-watcher.py):

Incident: orders-api canary deployment aborted automatically (revision 4)

Argo Rollouts staged revision 4 at 20 percent traffic weight and ran the orders-api-canary-health AnalysisTemplate against live Prometheus metrics. The p95-latency check failed on 2 of 3 sampled windows, exceeding the configured failureLimit of 1. Rollouts aborted the update automatically, scaled the canary ReplicaSet to zero, and reverted all traffic to the last known-good stable revision, with no manual rollback required.

Likely cause: the deployed revision introduced elevated response times (a test-injected FAIL_RATE flag), pushing roughly 60 percent of orders requests past 800ms and driving p95 latency well above the 500ms gate.

Suggested next step: confirm whether the regression was intentional (a load test) or accidental. If accidental, revert and re-run the pipeline; if intentional, exclude this build from automated promotion rather than letting it compete for the gate.

In production, scripts/rollout-watcher.py generates this automatically via the Claude API and files it as a GitHub Issue on every real abort. It is included in this repo but not wired to a live API key for this build, to avoid a paid dependency during a time-boxed portfolio project.
