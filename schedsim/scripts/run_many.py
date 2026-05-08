#!/usr/bin/python3

import string
import fire
import os
import sys
import subprocess
from multiprocessing import Process
from enum import Enum, auto
import csv
from pprint import pprint

# Only written for single Q
load_levels = [0.01, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99]
metrics = ['Count', 'Stolen', 'AVG (latency)', 'STDDev (latency)',
           '50th (latency)', '90th (latency)', '95th (latency)', '99th (latency)',
           'AVG (slowdown)', 'STDDev (slowdown)', '50th (slowdown)', '90th (slowdown)', 
           '95th (slowdown)', '99th (slowdown)', 'Reqs/time_unit']


def run(topo, mu, gen_type, proc_type, num_cores, quantum =5, stddev =0, ctxCost = 1):
    '''
    mu in us
    '''
    service_time_per_core_us = 1 / mu
    rps_capacity_per_core = 1 / service_time_per_core_us * 1000.0 * 1000.0
    total_rps_capacity = rps_capacity_per_core * num_cores
    injected_rps = [load_lvl * total_rps_capacity for load_lvl in load_levels]
    lambdas = [rps / 1000.0 / 1000.0 for rps in injected_rps]
    res_file = "out.txt"
    with open(res_file, 'w') as f:
        for l in lambdas:
            cmd = f"schedsim --topo={topo} --mu={mu} --genType={gen_type} --procType={proc_type} --lambda={l} --quantum={quantum} --stddev={stddev} --ctxCost={ctxCost}"
            print(f"Running... {cmd}")
            subprocess.run(cmd, stdout=f, shell=True)


def out_to_csv(input_f, output_f, stats):
    results = {}
    with open(input_f, 'r') as f:
        csv_reader = csv.reader(f, delimiter='\t')
        next_is_res = False
        correct_stats = False # Hack for now
        for row in csv_reader:
            if len(row) >= 3 and "interarrival_rate" in row[2]:
                load_lvl = (float(row[2].split(":")[1])/float(row[1].split(":")[1])/float(row[0].split(":")[1]))
                results[load_lvl] = {}
            if next_is_res:
                for i, metric in enumerate(metrics):
                    results[load_lvl][metric] = row[i]
            next_is_res = correct_stats and "Count" == row[0]
            correct_stats = len(row) == 1 and row[0].split(":")[1].strip() == stats

    with open(output_f, 'w') as f:
        writer = csv.writer(f, delimiter='\t')
        for load_lvl in results:
            # TODO: let user choose
            writer.writerow(
                [load_lvl, results[load_lvl]['50th (latency)'], results[load_lvl]['99th (latency)'], 
                results[load_lvl]['50th (slowdown)'], results[load_lvl]['99th (slowdown)']])


if __name__ == "__main__":
    fire.Fire({
        "run": run,
        "csv": out_to_csv
    })
