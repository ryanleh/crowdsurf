import argparse
import numpy as np
import pandas as pd
import re
import subprocess

# Parameters from the paper:
#
# Size (GB) | q = 2^32        |   q = 2^64       |
#           |  p  |  sqrt(N)  |   p    | sqrt(N) |
# ----------------------------------------------
# 0.25	      1186	 14331	   77749042	 8945
# 0.5	      997	 20519	   77749042	 12650
# 1	          997	 29366	   65378890	 18190
# 2	          838	 41529	   65378890	 25724
# 3	          838	 51515	   65378890	 31506
# 4	          838	 59484	   54976875	 36556
db_sizes_gb = [0.25, 0.5, 1, 2, 3, 4]
mod_ps = {
    32: [1186, 997,	997, 838, 838, 838],
    64: [77749042, 77749042, 65378890, 65378890, 65378890, 54976875],
}
sqrt_Ns = {
    32: [14331, 20519, 29366, 41529, 51515, 59484],
    64: [8945, 12650, 18190, 25724, 31506, 36556],
}

def run_bench(
    bench,
    db_size=None,
    log_q=None,
    p=None,
    sqrt_N=None,
    mode=None,
    hash_mode=None,
    packing="balanced",
    iters=None
):
    result = None
    try:
        cmd = [
            'go', 'run', '.', f'-bench={bench}',
            f'-packing={packing}',
            f'-q={log_q}' if log_q else None,
            f'-p={p}' if p else None,
            f'-rows={sqrt_N}' if sqrt_N else None,
            f'-cols={sqrt_N}' if sqrt_N else None,
            f'-mode={mode}' if mode else None,
            f'-hash={hash_mode}' if hash_mode else None,
            f'-test.benchtime={iters}x' if iters else None
        ]
        cmd = list(filter(None, cmd))
        print(f"Running command: {" ".join(cmd)}")
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=True,
        )
        # If `db_size` is passed then we're running an LHE query
        if db_size:
            row = {
                "size": db_size,
                "log_q": log_q,
                "p": p,
                "sqrt_N": sqrt_N,
            }
            return (result.stdout, row)
        else:
            return result.stdout
    except subprocess.CalledProcessError as e:
        print("Command failed")
        print("-----")
        print(f"STDOUT: {e.stdout}") 
        print(f"STDERR: {e.stderr}") 
        print("-----")
    except Exception as e:
        print(
            f"Command failed: {e}"
        )


def run_lhe_bench(query=True, rerun_lwe=False):
    # Create the dataframe tables to store results
    if query:
        bench_string = "query"
        metric = "time_ms"
        columns=["size", "log_q", "p", "sqrt_N", "time_ms"]
        # Returns the time in ms
        def parser(s, _):
            time_micros = float(re.search(r'(\d+\.\d+)µs', s).group(1))
            return time_micros / 1000.0
    else:
        bench_string = "preprocessing"
        metric = "tput_gb_s"
        columns=["size", "log_q", "p", "sqrt_N", "tput_gb_s"]
        # Returns the throughput in gb/s
        def parser(s, size):
            time_s = float(re.search(r'(\d+\.\d+)s', s).group(1))
            return size / time_s
    
    df_lwe = pd.DataFrame(columns=columns)
    df_hybrid = pd.DataFrame(columns=columns + ["improvement"])

    for log_q in [32, 64]:
        for (i, (p, sqrt_N)) in enumerate(zip(mod_ps[log_q], sqrt_Ns[log_q])):
            # We didn't benchmark 3GB for preprocessing
            if not query and db_sizes_gb[i] == 3:
                continue
            
            # 1) Run the hybrid benchmark
            (out, row) = run_bench(
                bench_string, db_sizes_gb[i], log_q, p, sqrt_N, "hybrid",
                iters=5
            )
            row[metric] = parser(out, db_sizes_gb[i])
            df_hybrid.loc[len(df_hybrid)] = row

            # 2) Run the standard LWE benchmark
            #
            # If not rerunning preprocessing, we only collect one sample
            if query or (rerun_lwe or (log_q==32 and i==0)):
                (out, row) = run_bench(
                    bench_string, db_sizes_gb[i], log_q, p, sqrt_N, "none"
                )
                row[metric] = parser(out, db_sizes_gb[i])
                df_lwe.loc[len(df_lwe)] = row
            else:
                # Copy the row from above here–we'll fill in the exact value
                # later
                row[metric] = np.nan
                df_lwe.loc[len(df_lwe)] = row

    # Compute improvements
    if not query and not rerun_lwe:
        # We use old benchmark numbers (which we calibrate) instead of
        # re-running
        old_pre_tput_s = {
            32: [0.001811725487, 0.001635269492, 0.001490312966, 0.001441618072, 0.001345297513],
            64: [0.0005891085609, 0.0005879240402, 0.00056891559, 0.0005692572616, 0.0005597631083],
        }

        # Compute the delta between the saved measurement and the
        # re-measured one
        delta = df_lwe.loc[0, "tput_gb_s"] / old_pre_tput_s[32][0] 

        # Input adjusted numbers to the saved dataframes
        for log_q in [32, 64]:
            filtered_df = df_lwe[df_lwe["log_q"] == log_q]
            for (i, (index, row)) in enumerate(filtered_df.iterrows()):
                if pd.isnull(row[metric]):
                    df_lwe.loc[index, metric] = old_pre_tput_s[log_q][i] * delta
    
    # Compute the change between the LWE and hybrid setups
    for i in range(len(df_hybrid)):
        improvement = df_hybrid.loc[i, metric] / df_lwe.loc[i, metric] 
        if query:
            improvement = 1 / improvement
        df_hybrid.loc[i, "improvement"] = improvement

    return (df_lwe, df_hybrid)

def run_batch_bench(batch_type="dpir"):
    # Create the dataframe tables to store results
    columns=["batch_size", "queries_per_sec", "upload_mb", "download_mb"]
    df = pd.DataFrame(columns=columns)

    if batch_type == "dpir":
        bench_string = "dpir"
        hash_mode = None
    elif batch_type == "cuckoo":
        bench_string = "pbc"
        hash_mode = "cuckoo"
    else:
        bench_string = "pbc"
        hash_mode = "hash"
        
    # Run the benchmark
    out = run_bench(bench_string, hash_mode=hash_mode, packing="storage")
    queries_per_sec = re.findall(r'Queries-per-sec.*: (\d+.\d+) Q/s', out)
    upload_mb = re.findall(r'Upload: (\d+.\d+)MB', out)
    download_mb = re.findall(r'Download: (\d+.\d+)MB', out)
    
    for (i, batch_size) in enumerate([8, 16, 24, 32, 40, 48, 56, 64]):
        row = {
            "batch_size": batch_size,
            "queries_per_sec": round(float(queries_per_sec[i]), 2),
            "upload_mb": round(float(upload_mb[i]), 2),
            "download_mb": round(float(download_mb[i]), 2),
        }
        df.loc[len(df)] = row

    return df


# TODO: Output nice graphs
if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '-b', '--batching', action='store_true',
        help="Run DPIR batching benchmarks"
    )
    parser.add_argument(
        '-p', '--preprocessing', action='store_true',
        help="Run LWE preprocessing benchmarks"
    )
    parser.add_argument(
        '-r', '--rerun-lwe', action='store_true',
        help="Rerun LWE preprocessing (very slow)"
    )
    parser.add_argument(
        '-q', '--query', action='store_true',
        help="Run LHE query benchmarks"
    )
    args = parser.parse_args()

    # TODO: Have switch for benches
    run_all = not (args.batching or args.preprocessing or args.query)

    if args.batching or run_all:
        df_dpir = run_batch_bench("dpir")
        print(f"\nDPIR batching:\n{df_dpir.to_markdown()}\n")
        
        df_hash = run_batch_bench("hash")
        print(f"\nHash batching:\n{df_hash.to_markdown()}\n")
        
        df_cuckoo = run_batch_bench("cuckoo")
        print(f"\nCuckoo batching:\n{df_cuckoo.to_markdown()}\n")

    if args.preprocessing or run_all:
        (df_pre_lwe, df_pre_hybrid) = run_lhe_bench(False, args.rerun_lwe)
        print(f"\nPlain LWE Preproc. Bench:\n{df_pre_lwe.to_markdown()}\n")
        print(f"Hybrid Preproc. Bench:\n{df_pre_hybrid.to_markdown()}\n")

    if args.query or run_all:
        (df_query_lwe, df_query_hybrid) = run_lhe_bench(True, False)
        print(f"\nPlain LWE Query Bench:\n{df_query_lwe.to_markdown()}\n")
        print(f"Hybrid Query Bench:\n{df_query_hybrid.to_markdown()}\n")


