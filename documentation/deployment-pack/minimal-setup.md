# Minimal Worker VM Installation
**Goal:** Join a hospital-side VM to a MicroK8s cluster as a **worker** using **Tailscale-only** networking (no public MicroK8s ports).  
**Security posture:** If host firewall policy is strict, keep inbound closed and allow only the minimum required traffic on `tailscale0`.  
---

## 0) Federation Variables (The federation administrator should provide these values)
- `TS_AUTHKEY` = short-lived, **one-off** Tailscale auth key (ideally tagged)
- `MASTER_TS` = the master VM's Tailscale IPv4 (e.g., `100.x.y.z`)
- `MICROK8S_JOIN_TOKEN` = the token used to join the microk8s cluster, **one-off**.

Example of values that should be provided:
```bash
export TS_AUTHKEY="tskey-auth-REDACTED"
export MASTER_TS="1.2.3.4"
export MICROK8S_JOIN_TOKEN="microk8s-join-token-REDACTED"
```

---

## 1) Install MicroK8s and Tailscale (worker VM)

```bash
sudo snap install microk8s --classic --channel=1.33/stable
curl -fsSL https://tailscale.com/install.sh | sh
```

---

## 2) Join the Tailnet (worker VM)

**Note:** `--accept-dns=false` prevents changing hospital DNS settings.

```bash
sudo tailscale up --auth-key="$TS_AUTHKEY" --accept-dns=false
tailscale ip -4
tailscale status
```

---

## 3) UFW minimum rules (explicit split: public network vs Tailnet)

Apply this section only if the hospital VM enforces UFW with restrictive defaults.
Assumed baseline policy: `sudo ufw default deny incoming` and a controlled outbound policy per hospital standard.

Rule scope in this section:
- **Public network rules**: not bound to `tailscale0`; used only so the Tailscale client can establish/maintain connectivity.
- **Tailnet-only rules**: explicitly bound to `tailscale0`; these carry MicroK8s node traffic.

### 3.1 Public network outbound rules (Tailscale client only)

```bash
# Public egress needed by the Tailscale daemon for control/relay/connectivity
sudo ufw allow out 443/tcp
sudo ufw allow out 80/tcp
sudo ufw allow out 3478/udp
sudo ufw allow out 41641/udp
```

**Important:** These are **public-network egress-only** exceptions. They are not MicroK8s service ports and they are not opened inbound.

### 3.2 Tailnet-only: worker -> master (MicroK8s join + API)

```bash
# Restricted to tailscale0 and to the master Tailscale IP
sudo ufw allow out on tailscale0 to "$MASTER_TS" port 25000 proto tcp   # cluster-agent
sudo ufw allow out on tailscale0 to "$MASTER_TS" port 16443 proto tcp   # Kubernetes API (MicroK8s)
```

### 3.3 Tailnet-only: master/cluster -> worker (kubelet secure port)

```bash
# Restricted to tailscale0 and master Tailscale IP
sudo ufw allow in on tailscale0 from "$MASTER_TS" to any port 10250 proto tcp
```

### 3.4 Tailnet-only: Calico VXLAN dataplane between nodes

```bash
# Calico VXLAN is UDP 4789 between nodes, only on tailscale0
sudo ufw allow in  on tailscale0 to any port 4789 proto udp
sudo ufw allow out on tailscale0 to any port 4789 proto udp
```

**Scope note:** VXLAN is node-to-node traffic, so `4789/udp` applies to all cluster node peers reachable on `tailscale0`, not only the master.
Use Tailscale ACLs/tags to ensure only authorized worker/master nodes can join that Tailnet segment.

**Dependency note:** These VXLAN rules are valid when the cluster backend is Calico VXLAN. If the backend changes (for example IPIP or WireGuard), ports/protocols must be adjusted by the cluster operator.

### 3.5 Reload and verify UFW rules

```bash
sudo ufw reload
sudo ufw status verbose
sudo ufw status numbered
```

---

## 4) Connectivity checks (before `microk8s join`)

```bash
tailscale ping "$MASTER_TS"

# Optional reachability checks to master
nc -vz -w2 "$MASTER_TS" 25000
nc -vz -w2 "$MASTER_TS" 16443
```

Expected:

* `tailscale ping` replies (no timeout)
* `nc` to 25000 and 16443 succeeds

---

## 5) Join MicroK8s as a worker

Run the join command provided securely by the master operator (short-lived / one-time):
```bash
sudo microk8s join $MASTER_TS:25000/$MICROK8S_JOIN_TOKEN --worker
```

If you get the error `Joining cluster failed. Could not verify the identity of 1.2.3.4. Use '--sk`
Please use the following command and notify the federation administrator:
```bash
sudo microk8s join $MASTER_TS:25000/$MICROK8S_JOIN_TOKEN --worker --skip-verify
```
