import re
import subprocess
import sys
import time

from pathlib import Path
    

# The following benchmark runs e2e benchmarks for several individual shards
# of the full CrowdSurf system to estimate costs.
#
# TODO: Explain full deployment + some of caveats of the estimation
#
# ---- System parameters ---- 
#

corr_wst = 0.01

# AWS cost of the various instances in dollars / hr
aws_dollars_per_hour = {
    "cpu": 0.36,
    "gpu": 3.1,
}

# Number of buckets
num_buckets = {
    "popular": 1,
    "full": 8,
    "baseline": 24,
}

# Bucket parameters: [queries, row, col, pMod]
bucket_params = {
    "popular": [8, 8, 164408, 593],
    "full": [3, 15, 596656, 419],
    "baseline": [2, 8, 380053, 498],
}

# Total number of shards: [hint, pir]
shards = {
    "popular": [1, 1],
    "full": [3, 16],
    "baseline": [6, 24]
}

def run_e2e(batch_size, rows, cols, p, bits=None, hint_ms=None):
    # First, ensure that the hint compression library is built
    cwd = Path.cwd()
    bazel_path = (cwd / "../external/hintless_pir").resolve()
    client_path = (cwd / "bin/client").resolve()

    try:
        result = subprocess.run(
            "bazel build -c opt //dpir:dpir_client".split(" "),
            capture_output=True,
            text=True,
            check=True,
            cwd=str(bazel_path.absolute()),
        )
    except subprocess.CalledProcessError as e:
        print(f"Attempt to build hint client failed: {e}")

    # Now run hint client in the background and PIR client
    hint_client = subprocess.Popen(
        "bazel run -c opt //dpir:dpir_client".split(" "),
        cwd=str(bazel_path.absolute()),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    # Run the e2e benchmark
    try:
        cmd = [
            "go", "run", ".", f"-rows={rows}", f"-cols={cols}",
            f"-batch_size={batch_size}", f"-p={p}",
            f"-hint_ms={hint_ms}" if hint_ms else None,
            f"-bits={bits}" if bits else None,
        ]
        cmd = list(filter(None, cmd))
        print(f"Running command: {' '.join(cmd)}")
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=True,
            cwd=str(client_path.absolute())
        )
    except subprocess.CalledProcessError as e:
        print(f"Error occured: {e.stderr}")
        print(f"Ensure that the PIR and Hint server are running.")
        sys.exit()

    # Kill the hint client
    hint_client.kill()

    # Extract the time for each component
    times_ms = re.search(
        r'.*Answer latency: (\d+\.\d+)ms \(p: (\d+\.\d+)ms, h: (\d+\.\d+)ms\)',
        result.stderr
    )
    batch = re.search(r'.*Batch Capacity: (\d+)', result.stderr).group(1)
    return [float(t) for t in times_ms.groups()] + [int(batch)]

def compute_metrics(shard_type, batch_cap, hint_latency_ms, pir_type="cpu", round_to=3):
    hint_cpu_s = hint_latency_ms / 1000 * shards[shard_type][0]
    hint_aws_cents = hint_cpu_s / 36 * aws_dollars_per_hour["cpu"]
    pir_clients_per = batch_cap / bucket_params[shard_type][0]
    
    pir_s = hint_latency_ms / 1000 * shards[shard_type][1] / pir_clients_per
    pir_aws_cents = pir_s / 36 * aws_dollars_per_hour[pir_type]

    return {
        "hint_cpu_s": round(hint_cpu_s, round_to),
        "hint_aws_cents": round(hint_aws_cents, round_to),
        "pir_s": round(pir_s, round_to),
        "pir_aws_cents": round(pir_aws_cents, round_to),
        "total_aws_cents": round(hint_aws_cents + pir_aws_cents, round_to),
    }

def pprint_dict(to_print, name):
    print(f"{name}: {{")
    for (k, v) in to_print.items():
        print(f"  {k:<15} : {v:>5}")
    print("}")

if __name__ == '__main__':
    # CrowdSurf has two different shard types corresponding to the popular or
    # full database (minus the number of entries that are stored locally)
    #
    # 1) Popular shard
    # --> This benchmark makes 8 queries to the hint compression machine (using
    #     all of its cores) and thus, is the number we should use for
    #     benchmarking the PIR batch capacity of the other shards (since they
    #     don't make 8 queries)

    ## TODO: Update bits
    _, _, hint_ms, batch_cap_pop = run_e2e(*bucket_params["popular"], bits=1)

    # 2) Full shard
    #
    # TODO: Add in option to add hint time
    _, _, _, batch_cap_full = run_e2e(
        *bucket_params["full"],
        hint_ms=hint_ms,
        bits=1
    )

    # The baseline has a single shard corresponding to the entire database
    _, _, _, batch_cap_base = run_e2e(
        *bucket_params["baseline"],
        hint_ms=hint_ms,
        bits=1
    )

    ## NOTE: These were the numbers we got when running this script
    #hint_ms = 529
    #batch_cap_pop = 1300
    #batch_cap_full = 275
    #batch_cap_base = 21

    # Compute baseline metrics
    baseline_metrics = compute_metrics("baseline", batch_cap_base, hint_ms)

    # Compute crowdsurf metrics, just a linear combination of `popular` and `full`
    popular_metrics = compute_metrics("popular", batch_cap_pop, hint_ms, "gpu", 15)
    full_metrics = compute_metrics("full", batch_cap_full, hint_ms, "gpu", 15)
    crowdsurf_metrics = {}
    for (k, v) in popular_metrics.items():
        crowdsurf_metrics[k] = round(corr_wst * full_metrics[k] + (1 - corr_wst) * v, 4)

    # Print stuff
    pprint_dict(baseline_metrics, "Baseline metrics")
    pprint_dict(crowdsurf_metrics, "CrowdSurf metrics")

    # TODO: Add in metrics for total time and communication
