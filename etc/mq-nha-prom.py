#!/usr/bin/env python3
import re, subprocess, sys

QMGR = "QM1"
#OUT_FILE = "/var/lib/node_exporter/textfile_collector/mq_nha.prom"
OUT_FILE="./mq_nha.prom"
def run_dspmq():
    # result = subprocess.run(
    #     ["dspmq", "-o", "nativeha", "-x", "-m", QMGR],
    #     capture_output=True, text=True
    # )
    # Test with podman
    result = subprocess.run(
        ["podman", "exec", "mq-node-1", "dspmq", "-o", "nativeha", "-x", "-m", QMGR],
        capture_output=True, text=True
    )
    return result.stdout

def parse(output):
    lines = output.splitlines()
    header = lines[0]

    # Group-level line: QMNAME / overall ROLE / QUORUM
    quorum_match = re.search(r"QUORUM\((\d+)/(\d+)\)", header)
    insync, total = quorum_match.groups() if quorum_match else (0, 0)

    instances = []
    for line in lines[1:]:
        if "INSTANCE(" not in line:
            continue
        m_inst     = re.search(r"INSTANCE\((\S+?)\)", line)
        m_role     = re.search(r"ROLE\((\S+?)\)", line)
        m_connactv = re.search(r"CONNACTV\((\S+?)\)", line)
        m_insync_i = re.search(r"INSYNC\((\S+?)\)", line)
        m_backlog  = re.search(r"BACKLOG\((\d+|\S*)\)", line)
        m_hastatus = re.search(r"HASTATUS\((\S+?)\)", line)
        if not (m_inst and m_role and m_connactv and m_insync_i and m_backlog and m_hastatus):
            continue
        inst     = m_inst.group(1)
        role     = m_role.group(1)
        connactv = m_connactv.group(1)
        insync_i = m_insync_i.group(1)
        backlog  = m_backlog.group(1)
        hastatus = m_hastatus.group(1)
        instances.append((inst, role, connactv, insync_i, backlog, hastatus))
    return insync, total, instances

def render(insync, total, instances):
    lines = []
    lines.append("# HELP ibmmq_nha_quorum_insync Number of in-sync instances")
    lines.append("# TYPE ibmmq_nha_quorum_insync gauge")
    lines.append(f'ibmmq_nha_quorum_insync{{qmgr="{QMGR}"}} {insync}')

    lines.append("# HELP ibmmq_nha_quorum_total Number of configured instances")
    lines.append("# TYPE ibmmq_nha_quorum_total gauge")
    lines.append(f'ibmmq_nha_quorum_total{{qmgr="{QMGR}"}} {total}')

    lines.append("# HELP ibmmq_nha_instance_role State-as-label for instance role")
    lines.append("# TYPE ibmmq_nha_instance_role gauge")
    for inst, role, connactv, insync_i, backlog, hastatus in instances:
        lines.append(f'ibmmq_nha_instance_role{{qmgr="{QMGR}",instance="{inst}",role="{role}"}} 1')

    lines.append("# HELP ibmmq_nha_instance_connected Connection active flag (1=yes,0=no)")
    lines.append("# TYPE ibmmq_nha_instance_connected gauge")
    for inst, role, connactv, insync_i, backlog, hastatus in instances:
        val = 1 if connactv.lower() == "yes" else 0
        lines.append(f'ibmmq_nha_instance_connected{{qmgr="{QMGR}",instance="{inst}"}} {val}')

    lines.append("# HELP ibmmq_nha_instance_insync In-sync flag (1=yes,0=no)")
    lines.append("# TYPE ibmmq_nha_instance_insync gauge")
    for inst, role, connactv, insync_i, backlog, hastatus in instances:
        val = 1 if insync_i.lower() == "yes" else 0
        lines.append(f'ibmmq_nha_instance_insync{{qmgr="{QMGR}",instance="{inst}"}} {val}')

    return "\n".join(lines) + "\n"

if __name__ == "__main__":
    raw = run_dspmq()
    insync, total, instances = parse(raw)
    text = render(insync, total, instances)
    with open(OUT_FILE, "w") as f:
        f.write(text)