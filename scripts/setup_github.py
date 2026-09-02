#!/usr/bin/env python3
# setup_github.py - Universal GitHub CI/CD setup for Go K8s deployments
# Sets up all required secrets and variables in one batch

import base64
import json
import os
import sys
import urllib.request
import urllib.error
from typing import Dict, Any, Optional


class GitHubSetup:
    def __init__(self, repo: str, token: str):
        self.repo = repo
        self.token = token
        self.base_url = f"https://api.github.com/repos/{repo}"
        self.headers = {
            "Authorization": f"token {token}",
            "Accept": "application/vnd.github.v3+json",
            "Content-Type": "application/json",
        }
        self._public_key: Optional[Dict[str, str]] = None

    def _request(self, method: str, endpoint: str, data: Optional[Dict] = None) -> Dict:
        """Make GitHub API request."""
        url = f"{self.base_url}{endpoint}"
        req = urllib.request.Request(
            url,
            data=json.dumps(data).encode() if data else None,
            headers=self.headers,
            method=method,
        )
        try:
            with urllib.request.urlopen(req) as response:
                return json.loads(response.read().decode()) if response.status != 204 else {}
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return {}
            raise

    def get_public_key(self) -> Dict[str, str]:
        """Get repository public key for secret encryption."""
        if self._public_key is None:
            self._public_key = self._request("GET", "/actions/secrets/public-key")
        return self._public_key

    def encrypt_secret(self, value: str) -> str:
        """Encrypt secret using repository public key."""
        try:
            from nacl import encoding, public
        except ImportError:
            raise RuntimeError("PyNaCl is required. Install with: pip install PyNaCl")

        key_info = self.get_public_key()
        pubkey = public.PublicKey(key_info["key"].encode(), encoding.Base64Encoder())
        sealed_box = public.SealedBox(pubkey)
        encrypted = sealed_box.encrypt(value.encode())
        return base64.b64encode(encrypted).decode()

    def set_secret(self, name: str, value: str) -> bool:
        """Set a GitHub Actions secret."""
        try:
            encrypted = self.encrypt_secret(value)
            key_id = self.get_public_key()["key_id"]
            self._request("PUT", f"/actions/secrets/{name}", {
                "encrypted_value": encrypted,
                "key_id": key_id,
            })
            print(f"✓ Secret: {name}")
            return True
        except Exception as e:
            print(f"✗ Secret {name}: {e}")
            return False

    def set_variable(self, name: str, value: str) -> bool:
        """Set a GitHub Actions variable."""
        try:
            # Try to create new variable
            self._request("POST", "/actions/variables", {
                "name": name,
                "value": value,
            })
            print(f"✓ Variable: {name}")
            return True
        except urllib.error.HTTPError as e:
            if e.code == 409:  # Conflict - variable exists
                try:
                    self._request("PATCH", f"/actions/variables/{name}", {
                        "value": value,
                    })
                    print(f"✓ Variable (updated): {name}")
                    return True
                except Exception as e2:
                    print(f"✗ Variable {name}: {e2}")
                    return False
            print(f"✗ Variable {name}: {e}")
            return False

    def setup_from_config(self, config: Dict[str, Any]) -> bool:
        """Set up all secrets and variables from configuration."""
        print(f"Setting up GitHub CI/CD for {self.repo}...")

        # Extract values from config
        cluster = config.get("cluster", {})
        app_name = config.get("app_name", "")
        namespace = config.get("namespace", "")
        domain = config.get("domain", "")
        registry = config.get("registry", "ghcr.io")

        # Secrets (sensitive)
        secrets = {
            "K8S_CLIENT_CERT": os.environ.get("K8S_CLIENT_CERT", ""),
            "K8S_CLIENT_KEY": os.environ.get("K8S_CLIENT_KEY", ""),
            "K8S_CA_CERT": os.environ.get("K8S_CA_CERT", ""),
            "INGRESS_CA_CERT": os.environ.get("K8S_CA_CERT", ""),
        }

        # Variables (non-sensitive)
        api_server = cluster.get("api_server", "")
        host_port = api_server.replace("https://", "").split(":")
        variables = {
            "K8S_NAMESPACE": namespace,
            "APP_NAME": app_name,
            "DOMAIN": domain,
            "K8S_HOST": host_port[0] if len(host_port) > 0 else "",
            "K8S_PORT": host_port[1] if len(host_port) > 1 else "443",
            "K8S_EXPECTED_API_SERVER": api_server,
            "K8S_EXPECTED_CA_SHA256": cluster.get("ca_sha256", ""),
            "INGRESS_ENDPOINT": cluster.get("traefik_ip", ""),
            "REGISTRY": registry,
            "GO_VERSION": config.get("go", {}).get("version", "1.22"),
        }

        # Set all secrets
        success = True
        print("\n--- Setting Secrets ---")
        for name, value in secrets.items():
            if value:
                if not self.set_secret(name, value):
                    success = False
            else:
                print(f"⚠ Skipping empty secret: {name}")

        # Set all variables
        print("\n--- Setting Variables ---")
        for name, value in variables.items():
            if value:
                if not self.set_variable(name, value):
                    success = False
            else:
                print(f"⚠ Skipping empty variable: {name}")

        return success


def main():
    # Get configuration from environment or command line
    repo = os.environ.get("REPO") or (sys.argv[1] if len(sys.argv) > 1 else None)
    token = os.environ.get("GITHUB_TOKEN")

    if not repo:
        print("Error: REPO environment variable or first argument required")
        sys.exit(1)
    if not token:
        print("Error: GITHUB_TOKEN environment variable required")
        sys.exit(1)

    # Read config from stdin or file
    config_file = os.environ.get("CONFIG_FILE", "/tmp/preflight-config.json")
    if os.path.exists(config_file):
        with open(config_file) as f:
            config = json.load(f)
    else:
        # Read from stdin
        config = json.load(sys.stdin)

    # Setup GitHub
    setup = GitHubSetup(repo, token)
    success = setup.setup_from_config(config)

    if success:
        print("\n✓ GitHub CI/CD setup complete")
        sys.exit(0)
    else:
        print("\n✗ Some items failed to set")
        sys.exit(1)


if __name__ == "__main__":
    main()
