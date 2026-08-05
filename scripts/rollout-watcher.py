#!/usr/bin/env python3
import json, subprocess, urllib.request, sys, pathlib

STATE_FILE = pathlib.Path.home() / ".orderpulse-last-incident"
SECRETS_FILE = pathlib.Path.home() / ".orderpulse-secrets"

def run(cmd):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True).stdout.strip()

def load_api_key():
    if not SECRETS_FILE.exists():
        print("No secrets file at", SECRETS_FILE)
        sys.exit(1)
    for line in SECRETS_FILE.read_text().splitlines():
        if line.startswith("ANTHROPIC_API_KEY="):
            return line.split("=", 1)[1].strip()
    print("ANTHROPIC_API_KEY not found in secrets file")
    sys.exit(1)

def main():
    phase = run("kubectl get rollout orders-api -n default -o jsonpath='{.status.phase}'")
    if phase != "Degraded":
        return

    message = run("kubectl get rollout orders-api -n default -o jsonpath='{.status.message}'")
    last = STATE_FILE.read_text().strip() if STATE_FILE.exists() else ""
    if message == last:
        return

    analysis = run("kubectl get analysisrun -n default -o json")
    logs = run("kubectl logs -n default -l app=orders-api --tail=40 --prefix=true")

    prompt = f"""You are an SRE assistant. A Kubernetes canary deployment (Argo Rollouts) for a service called orders-api was just automatically aborted.

Rollout status message:
{message}

Recent AnalysisRun objects (JSON, may include multiple runs):
{analysis[:3000]}

Recent pod logs (last 40 lines across all orders-api pods):
{logs[:2000]}

Write a concise incident summary (under 200 words) for a GitHub issue: what happened, which metric(s) failed and by how much if visible, likely cause, and one suggested next step. Plain text, no markdown headers."""

    api_key = load_api_key()
    body = json.dumps({
        "model": "claude-haiku-4-5-20251001",
        "max_tokens": 500,
        "messages": [{"role": "user", "content": prompt}]
    }).encode()

    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=body,
        headers={
            "content-type": "application/json",
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01"
        }
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        data = json.loads(resp.read())
    summary = "".join(b["text"] for b in data["content"] if b["type"] == "text")

    issue_title = "[auto] orders-api canary aborted"
    issue_body = f"{summary}\n\n---\nRaw status message: {message}"
    subprocess.run(["gh", "issue", "create", "--title", issue_title, "--body", issue_body], check=True)

    STATE_FILE.write_text(message)
    print("Incident issue created.")

if __name__ == "__main__":
    main()
