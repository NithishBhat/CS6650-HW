"""
Generate HW10 PDF report.
python generate_report.py
"""

from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.lib.colors import HexColor
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, Image, Table, TableStyle,
    PageBreak,
)
from reportlab.lib import colors
import os

OUTPUT = "report.pdf"
GRAPHS = "graphs"

def build_report():
    doc = SimpleDocTemplate(
        OUTPUT, pagesize=letter,
        leftMargin=0.75*inch, rightMargin=0.75*inch,
        topMargin=0.75*inch, bottomMargin=0.75*inch,
    )

    styles = getSampleStyleSheet()

    title_style = ParagraphStyle(
        "CustomTitle", parent=styles["Title"], fontSize=24,
        spaceAfter=6, textColor=HexColor("#1a1a2e"),
    )
    subtitle_style = ParagraphStyle(
        "Subtitle", parent=styles["Normal"], fontSize=14,
        spaceAfter=4, textColor=HexColor("#555555"), alignment=TA_CENTER,
    )
    h1 = ParagraphStyle(
        "H1", parent=styles["Heading1"], fontSize=18,
        spaceBefore=20, spaceAfter=10, textColor=HexColor("#16213e"),
        borderWidth=1, borderColor=HexColor("#16213e"), borderPadding=4,
    )
    h2 = ParagraphStyle(
        "H2", parent=styles["Heading2"], fontSize=14,
        spaceBefore=14, spaceAfter=8, textColor=HexColor("#0f3460"),
    )
    h3 = ParagraphStyle(
        "H3", parent=styles["Heading3"], fontSize=12,
        spaceBefore=10, spaceAfter=6, textColor=HexColor("#533483"),
    )
    body = ParagraphStyle(
        "Body", parent=styles["Normal"], fontSize=10,
        spaceAfter=8, leading=14, alignment=TA_JUSTIFY,
    )
    bullet = ParagraphStyle(
        "Bullet", parent=body, leftIndent=20, bulletIndent=10,
        spaceBefore=2, spaceAfter=2,
    )

    story = []

    def add_spacer(h=0.15):
        story.append(Spacer(1, h * inch))

    def p(text, style=body):
        story.append(Paragraph(text, style))

    def add_graph(filename, w=6.5, h=2.5):
        path = os.path.join(GRAPHS, filename)
        if os.path.exists(path):
            story.append(Image(path, width=w*inch, height=h*inch))
            add_spacer(0.1)

    def add_table(data, col_widths=None):
        t = Table(data, colWidths=col_widths, hAlign="LEFT")
        t.setStyle(TableStyle([
            ("BACKGROUND", (0, 0), (-1, 0), HexColor("#16213e")),
            ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
            ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
            ("FONTSIZE", (0, 0), (-1, -1), 9),
            ("ALIGN", (1, 0), (-1, -1), "CENTER"),
            ("ALIGN", (0, 0), (0, -1), "LEFT"),
            ("GRID", (0, 0), (-1, -1), 0.5, HexColor("#cccccc")),
            ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, HexColor("#f8f8f8")]),
            ("TOPPADDING", (0, 0), (-1, -1), 4),
            ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
            ("LEFTPADDING", (0, 0), (-1, -1), 6),
            ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ]))
        story.append(t)
        add_spacer(0.15)

    # === HOW IT WORKS ===
    p("1. How It Works", h1)

    p("1.1 Leader-Follower", h2)
    p("One leader (node0) accepts all writes, replicates to 4 followers sequentially with simulated delays (200ms between sends, 100ms on each follower). "
      "Reads go to any node; R=1 reads locally, R>1 gathers from peers (50ms delay each).", body)

    p("1.2 Leaderless", h2)
    p("Any node can accept writes and becomes the Write Coordinator for that request. "
      "Replication works the same way as leader mode, just without a fixed leader.", body)

    p("1.3 W/R Configs", h2)

    add_table([
        ["Config", "W", "R", "W+R", "Consistent?", "Tradeoff"],
        ["W=5, R=1", "5", "1", "6 > 5", "Yes", "Slow writes, instant reads"],
        ["W=1, R=5", "1", "5", "6 > 5", "Yes*", "Fast writes, slow reads"],
        ["W=3, R=3", "3", "3", "6 > 5", "Yes", "Balanced"],
        ["Leaderless", "5", "1", "6 > 5", "Yes", "Slow writes, instant reads"],
    ], col_widths=[1.3*inch, 0.5*inch, 0.5*inch, 0.7*inch, 1*inch, 2.5*inch])

    p("<i>*W=1 R=5 showed stale reads since the write returns before replication finishes.</i>", body)

    p("1.4 Version Numbers", h2)
    p("Each key has an incrementing version; replicas only accept higher versions. "
      "Load tester tracks max version per key to detect stale reads.", body)

    story.append(PageBreak())

    # === IMPLEMENTATION DETAILS ===
    p("2. N/R/W Implementation Details", h1)

    p("2.1 Leader W=5, R=1", h2)
    p("<b>Write:</b> Leader saves locally, then POSTs to each follower sequentially (200ms gap, 100ms follower delay). Blocks until all 4 ack. ~1200ms total.", body)
    p("<b>Read:</b> Local map lookup. ~3ms.", body)

    p("2.2 Leader W=1, R=5", h2)
    p("<b>Write:</b> Leader saves locally and returns 201 immediately. Goroutine replicates in background. ~1ms client latency.", body)
    p("<b>Read:</b> Queries all 5 nodes in parallel (50ms peer delay), returns highest version. ~54ms.", body)

    p("2.3 Leader W=3, R=3", h2)
    p("<b>Write:</b> Leader saves locally + waits for 2 followers (quorum of 3). Remaining 2 replicated in background. ~600ms.", body)
    p("<b>Read:</b> Queries peers in parallel, needs 3 total responses, returns highest version. ~55ms.", body)

    p("2.4 Leaderless W=5, R=1", h2)
    p("<b>Write:</b> Receiving node becomes coordinator, replicates to all 4 others sequentially, waits for all. ~1250ms.", body)
    p("<b>Read:</b> Local map lookup. ~4ms.", body)

    story.append(PageBreak())

    # === LOAD TEST RESULTS ===
    p("3. Load Test Results", h1)

    p("10 users, 60 seconds per run, 10-key pool for temporal locality.", body)

    p("3.1 Summary", h2)

    add_table([
        ["Config", "Ratio", "Read Avg", "Read p50", "Read p99", "Write Avg", "Write p50", "Write p99", "Stale", "Total Reqs"],
        ["Leader W=5 R=1", "1/99", "3.1ms", "3ms", "8ms", "1211ms", "1200ms", "1200ms", "0", "11,728"],
        ["Leader W=5 R=1", "10/90", "2.9ms", "3ms", "7ms", "1211ms", "1200ms", "1200ms", "0", "3,598"],
        ["Leader W=5 R=1", "50/50", "3.2ms", "3ms", "9ms", "1212ms", "1200ms", "1200ms", "0", "889"],
        ["Leader W=5 R=1", "90/10", "6.0ms", "6ms", "17ms", "1212ms", "1200ms", "1200ms", "0", "510"],
        ["Leader W=1 R=5", "1/99", "54.2ms", "54ms", "60ms", "40ms", "45ms", "52ms", "0", "6,670"],
        ["Leader W=1 R=5", "10/90", "53.8ms", "53ms", "59ms", "23ms", "7ms", "50ms", "0", "6,957"],
        ["Leader W=1 R=5", "50/50", "53.9ms", "54ms", "59ms", "46ms", "45ms", "52ms", "5", "7,018"],
        ["Leader W=1 R=5", "90/10", "53.6ms", "53ms", "60ms", "46ms", "45ms", "52ms", "1", "7,347"],
        ["Leader W=3 R=3", "1/99", "55.1ms", "55ms", "62ms", "608ms", "610ms", "620ms", "0", "6,091"],
        ["Leader W=3 R=3", "10/90", "54.6ms", "54ms", "60ms", "608ms", "610ms", "630ms", "0", "4,134"],
        ["Leader W=3 R=3", "50/50", "54.6ms", "54ms", "61ms", "608ms", "610ms", "610ms", "0", "1,672"],
        ["Leader W=3 R=3", "90/10", "56.8ms", "55ms", "80ms", "608ms", "610ms", "620ms", "0", "1,020"],
        ["Leaderless W=5 R=1", "1/99", "4.6ms", "4ms", "11ms", "1252ms", "1300ms", "1300ms", "0", "10,864"],
        ["Leaderless W=5 R=1", "10/90", "3.4ms", "3ms", "8ms", "1248ms", "1300ms", "1300ms", "0", "3,583"],
        ["Leaderless W=5 R=1", "50/50", "3.4ms", "3ms", "10ms", "1231ms", "1200ms", "1300ms", "0", "893"],
        ["Leaderless W=5 R=1", "90/10", "4.0ms", "4ms", "11ms", "1216ms", "1200ms", "1300ms", "0", "512"],
    ], col_widths=[1.1*inch, 0.5*inch, 0.6*inch, 0.6*inch, 0.6*inch, 0.6*inch, 0.6*inch, 0.6*inch, 0.45*inch, 0.7*inch])

    p("3.2 Throughput (requests/sec)", h2)

    add_table([
        ["Config", "Ratio", "Read req/s", "Write req/s", "Total req/s"],
        ["Leader W=5 R=1", "1/99", "195.8", "2.0", "197.8"],
        ["Leader W=5 R=1", "10/90", "54.3", "6.2", "60.5"],
        ["Leader W=5 R=1", "50/50", "7.3", "7.6", "15.0"],
        ["Leader W=5 R=1", "90/10", "0.8", "7.9", "8.6"],
        ["Leader W=1 R=5", "1/99", "111.3", "1.3", "112.6"],
        ["Leader W=1 R=5", "10/90", "106.0", "11.4", "117.3"],
        ["Leader W=1 R=5", "50/50", "59.5", "58.9", "118.4"],
        ["Leader W=1 R=5", "90/10", "12.3", "111.5", "123.8"],
        ["Leader W=3 R=3", "1/99", "101.4", "1.4", "102.7"],
        ["Leader W=3 R=3", "10/90", "63.1", "6.6", "69.7"],
        ["Leader W=3 R=3", "50/50", "14.9", "13.2", "28.2"],
        ["Leader W=3 R=3", "90/10", "2.1", "15.0", "17.1"],
        ["Leaderless W=5 R=1", "1/99", "181.2", "2.1", "183.4"],
        ["Leaderless W=5 R=1", "10/90", "54.2", "6.0", "60.2"],
        ["Leaderless W=5 R=1", "50/50", "7.5", "7.5", "15.0"],
        ["Leaderless W=5 R=1", "90/10", "0.8", "7.9", "8.6"],
    ], col_widths=[1.3*inch, 0.7*inch, 1*inch, 1*inch, 1*inch])

    p("3.3 Stale Reads", h2)

    add_table([
        ["Config", "1/99", "10/90", "50/50", "90/10", "Total"],
        ["Leader W=5 R=1", "0", "0", "0", "0", "0"],
        ["Leader W=1 R=5", "0", "0", "5", "1", "6"],
        ["Leader W=3 R=3", "0", "0", "0", "0", "0"],
        ["Leaderless W=5 R=1", "0", "0", "0", "0", "0"],
    ], col_widths=[1.5*inch, 0.8*inch, 0.8*inch, 0.8*inch, 0.8*inch, 0.8*inch])

    story.append(PageBreak())

    # === GRAPHS ===
    p("4. Graphs", h1)

    p("4.1 Latency Distributions", h2)

    p("Leader W=5, R=1:", body)
    add_graph("leader_w5r1_latency.png", w=6.5, h=2.4)

    p("Leader W=1, R=5:", body)
    add_graph("leader_w1r5_latency.png", w=6.5, h=2.4)

    p("Leader W=3, R=3:", body)
    add_graph("leader_w3r3_latency.png", w=6.5, h=2.4)

    story.append(PageBreak())

    p("Leaderless W=5, R=1:", body)
    add_graph("leaderless_w5r1_latency.png", w=6.5, h=2.4)

    p("4.2 Stale Reads", h2)
    add_graph("stale_reads.png", w=6.5, h=2.4)

    p("4.3 Average Latency Comparison", h2)
    add_graph("avg_latency_summary.png", w=6.5, h=2.6)

    p("4.4 Throughput", h2)
    add_graph("throughput.png", w=6.5, h=2.6)

    p("4.5 Read-Write Interval Distribution", h2)
    p("Time gap between a write and the next read to the same key. "
      "Small intervals mean the load tester is reading keys shortly after writing them, which is what we need to catch staleness.", body)
    add_graph("rw_intervals.png", w=6.5, h=4.0)

    story.append(PageBreak())

    # === BRIEF ANALYSIS ===
    p("5. Analysis", h1)

    p("<b>Leader W=5, R=1</b>", h3)
    p("- Writes fixed at ~1211ms (4 x 300ms sequential replication). Reads ~3ms (local lookup).", bullet)
    p("- 0 stale reads; all nodes updated before client gets 201.", bullet)
    p("- Throughput drops 23x from read-heavy to write-heavy (198 req/s to 8.5 req/s) because writes block for 1.2s.", bullet)
    p("- Best for read-heavy workloads like product catalogs.", bullet)

    p("<b>Leader W=1, R=5</b>", h3)
    p("- Writes ~23-46ms (local save, background replication). Reads ~54ms (50ms peer sleep floor).", bullet)
    p("- 6 stale reads at higher write ratios; async replication races with the read gather.", bullet)
    p("- Throughput stays stable across ratios (~112-124 req/s) since both reads and writes are cheap.", bullet)
    p("- Best for write-heavy or balanced workloads.", bullet)

    p("<b>Leader W=3, R=3</b>", h3)
    p("- Writes ~608ms (half of W=5; only needs 2 followers). Reads ~55ms (same 50ms floor).", bullet)
    p("- 0 stale reads; W+R=6 > N=5 guarantees quorum overlap.", bullet)
    p("- Good middle ground when you need consistency without 1.2s writes.", bullet)

    p("<b>Leaderless W=5, R=1</b>", h3)
    p("- Similar to Leader W=5 R=1 but writes ~30-40ms slower (coordinator varies, less connection reuse).", bullet)
    p("- 0 stale reads; W=N ensures full replication before ack.", bullet)
    p("- Trade-off vs leader: no single point of failure, but slightly worse write latency.", bullet)

    p("<b>Test generator</b>: 10 Locust users pick from a pool of 10 keys randomly. "
      "Small pool means reads and writes frequently hit the same key within short time windows.", body)

    doc.build(story)
    print(f"Report generated: {OUTPUT}")


if __name__ == "__main__":
    build_report()
