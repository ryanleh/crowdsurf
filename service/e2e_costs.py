# TODO: Add explanation of evaluation design

#
# ---- NOTE: Parameters to replace after running experiments---- 
#

# The latency of hint compression for the vario
hint_latency_ms = {
    "popular": 526,
    "full": 475,
    "baseline": 529,
}

# The total number of PIR queries the server can process in the _same_ amount
# of time it takes to service the hint compression
pir_batch = {
    "popular": 1300, # GPU
    "full": 275, # GPU
    "baseline": 21, # CPU
}

#
# ---- Static parameters ---- 
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

# Queries to each bucket
bucket_queries = {
    "popular": 8,
    "full": 3,
    "baseline": 2,
}

# Bucket parameters: [row, col, pMod]
bucket_params = {
    "popular": [8, 164408, 593],
    "full": [15, 596656, 419],
    "baseline": [8, 380053, 498],
}

# Total number of shards: [hint, pir]
shards = {
    "popular": [1, 1],
    "full": [3, 16],
    "baseline": [6, 24]
}

def compute_metrics(shard_type, pir_type="cpu", round_to=3):
    hint_cpu_s = hint_latency_ms[shard_type] / 1000 * shards[shard_type][0]
    hint_aws_cents = hint_cpu_s / 36 * aws_dollars_per_hour["cpu"]
    pir_clients_per = pir_batch[shard_type] / bucket_queries[shard_type]
    
    pir_cpu_s = hint_latency_ms[shard_type] / 1000 * shards[shard_type][1] / pir_clients_per
    pir_aws_cents = pir_cpu_s / 36 * aws_dollars_per_hour[pir_type]

    return {
        "hint_cpu_s": round(hint_cpu_s, round_to),
        "hint_aws_cents": round(hint_aws_cents, round_to),
        "pir_cpu_s": round(pir_cpu_s, round_to),
        "pir_aws_cents": round(pir_aws_cents, round_to),
        "total_aws_cents": round(hint_aws_cents + pir_aws_cents, round_to),
    }

def pprint_dict(to_print, name):
    print(f"{name}: {{")
    for (k, v) in to_print.items():
        print(f"  {k:<15} : {v:>5}")
    print("}")

# Compute baseline metrics
baseline_metrics = compute_metrics("baseline")

# Compute crowdsurf metrics, just a linear combination of `popular` and `full`
popular_metrics = compute_metrics("popular", "gpu", 15)
full_metrics = compute_metrics("full", "gpu", 15)
crowdsurf_metrics = {}
for (k, v) in popular_metrics.items():
    crowdsurf_metrics[k] = round(corr_wst * full_metrics[k] + (1 - corr_wst) * v, 4)

# Print stuff
pprint_dict(baseline_metrics, "Baseline metrics")
pprint_dict(crowdsurf_metrics, "CrowdSurf metrics")
