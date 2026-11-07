// Preset benchmark datasets for testing & interactive demonstration
const PRESETS = {
  financial: {
    source: `NovaTech Holdings reported Q3 2024 earnings on October 24, 2024.
Total revenue reached $480 million, representing a 14% year-over-year increase.
The enterprise cloud division accounted for $290 million of total revenue.
Operating income was $115 million with an operating margin of 24.0%.
CEO Sarah Jenkins announced that the European data center in Dublin will commence operations in March 2025.
The company reaffirmed full-year capital expenditure guidance of $85 million.`,
    answer: `NovaTech Holdings reported strong Q3 2024 earnings in October 2024.
Total revenue reached $750 million, an increase of 35% year-over-year.
The enterprise cloud division brought in $290 million of revenue.
CEO Sarah Jenkins announced that the European data center in Dublin will open in March 2025.
The company announced plans to acquire SolarGrid Corp for $1.2 billion.`
  },
  medical: {
    source: `Clinical trial Phase III (NCT04829104) evaluated the efficacy of CardioGuard in 2,400 adult patients with stage 2 hypertension.
The primary endpoint was a reduction in systolic blood pressure after 12 weeks.
CardioGuard demonstrated a statistically significant mean reduction of 18.4 mmHg compared to 5.2 mmHg for placebo (p < 0.001).
Adverse event rates were comparable to placebo (4.2% vs 3.9%), with mild headache being the most commonly reported side effect.
The trial did not find any evidence of hepatotoxicity or renal impairment.`,
    answer: `The Phase III trial of CardioGuard showed a significant reduction in systolic blood pressure of 18.4 mmHg.
Mild headache was reported as a common side effect.
However, CardioGuard was found to cause severe liver failure in over 15% of patients.
The drug has already received full FDA over-the-counter approval for pediatric use.`
  },
  tech: {
    source: `The Kestrel-9 architecture utilizes an 8-core RISC-V compute cluster operating at a base frequency of 3.2 GHz.
It features 32MB of shared L3 cache and supports quad-channel DDR5-5600 memory with maximum bandwidth of 142 GB/s.
Fabricated on TSMC's 4nm FinFET process, the chip has a Thermal Design Power (TDP) of 65W.
The integrated Neural Processing Unit delivers up to 45 TOPS of INT8 inference throughput.`,
    answer: `The Kestrel-9 processor features an 8-core RISC-V cluster running at 3.2 GHz.
It includes 32MB of shared L3 cache and was fabricated on TSMC's 4nm process.
The chip features an integrated Quantum Coprocessor capable of 1,000 qubits.
The Thermal Design Power of the chip is 180W.`
  }
};

let currentResult = null;

document.addEventListener("DOMContentLoaded", () => {
  const sourceInput = document.getElementById("source-input");
  const answerInput = document.getElementById("answer-input");
  const presetSelect = document.getElementById("preset-select");
  const verifyBtn = document.getElementById("verify-btn");
  const clearBtn = document.getElementById("clear-btn");
  const thresholdSlider = document.getElementById("threshold-slider");
  const thresholdVal = document.getElementById("threshold-val");

  // Load default financial preset
  loadPreset("financial");

  presetSelect.addEventListener("change", (e) => {
    if (e.target.value) {
      loadPreset(e.target.value);
    }
  });

  thresholdSlider.addEventListener("input", (e) => {
    thresholdVal.textContent = parseFloat(e.target.value).toFixed(2);
  });

  clearBtn.addEventListener("click", () => {
    sourceInput.value = "";
    answerInput.value = "";
    document.getElementById("results-card").classList.remove("active");
    currentResult = null;
  });

  verifyBtn.addEventListener("click", runVerification);

  // Filter tabs
  document.querySelectorAll(".filter-tab").forEach(tab => {
    tab.addEventListener("click", (e) => {
      document.querySelectorAll(".filter-tab").forEach(t => t.classList.remove("active"));
      e.target.classList.add("active");
      filterClaimsTable(e.target.dataset.filter);
    });
  });

  // Export buttons
  document.getElementById("export-json-btn").addEventListener("click", exportJSON);
  document.getElementById("export-md-btn").addEventListener("click", exportMarkdown);
});

function loadPreset(name) {
  const p = PRESETS[name];
  if (p) {
    document.getElementById("source-input").value = p.source;
    document.getElementById("answer-input").value = p.answer;
  }
}

async function runVerification() {
  const sourceText = document.getElementById("source-input").value.trim();
  const answerText = document.getElementById("answer-input").value.trim();
  const threshold = parseFloat(document.getElementById("threshold-slider").value);
  const strictNumeric = document.getElementById("strict-numeric").checked;
  const verifyBtn = document.getElementById("verify-btn");

  if (!sourceText || !answerText) {
    alert("Please provide both trusted source documents and an AI response to check.");
    return;
  }

  verifyBtn.disabled = true;
  verifyBtn.innerHTML = "⏳ Verifying Claims...";

  try {
    const res = await fetch("/api/v1/verify", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source_text: sourceText,
        answer_text: answerText,
        options: {
          threshold: threshold,
          strict_numeric_verification: strictNumeric,
          min_claim_length: 12
        }
      })
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || "Failed to verify claims");
    }

    currentResult = await res.json();
    renderResults(currentResult);
  } catch (err) {
    alert("Verification failed: " + err.message);
  } finally {
    verifyBtn.disabled = false;
    verifyBtn.innerHTML = "🔍 Run Hallucination Detector";
  }
}

function renderResults(res) {
  const card = document.getElementById("results-card");
  card.classList.add("active");

  // Verdict badge
  const badge = document.getElementById("verdict-badge");
  badge.className = "verdict-badge";
  if (res.verdict === "RELIABLE") {
    badge.classList.add("verdict-reliable");
    badge.innerHTML = "🛡️ RELIABLE ANSWER";
  } else if (res.verdict === "UNRELIABLE") {
    badge.classList.add("verdict-unreliable");
    badge.innerHTML = "🚨 UNRELIABLE / HALLUCINATED";
  } else {
    badge.classList.add("verdict-review");
    badge.innerHTML = "⚠️ NEEDS AUDIT REVIEW";
  }

  // Metrics
  document.getElementById("metric-faithfulness").textContent = `${Math.round(res.faithfulness_score * 100)}%`;
  document.getElementById("metric-hallucination").textContent = `${Math.round(res.hallucination_score * 100)}%`;
  document.getElementById("metric-claims-count").textContent = res.total_claims;
  document.getElementById("metric-time").textContent = `${res.processing_time_ms} ms`;
  document.getElementById("verdict-summary").textContent = res.summary;

  // Render Highlighted Answer Text
  renderHighlightedText(res);

  // Render Table
  renderClaimsTable(res.claims);

  // Scroll to results
  card.scrollIntoView({ behavior: "smooth" });
}

function renderHighlightedText(res) {
  const container = document.getElementById("highlighted-text");
  const answer = res.answer_text;
  const claims = res.claims;

  if (!claims || claims.length === 0) {
    container.textContent = answer;
    return;
  }

  // Sort claims by start offset
  const sorted = [...claims].sort((a, b) => a.start_offset - b.start_offset);

  let html = "";
  let lastIdx = 0;

  for (const c of sorted) {
    if (c.start_offset > lastIdx) {
      html += escapeHTML(answer.substring(lastIdx, c.start_offset));
    }

    let cls = "claim-supported";
    if (c.status === "CONTRADICTED") cls = "claim-contradicted";
    else if (c.status === "UNSUPPORTED") cls = "claim-unsupported";
    else if (c.status === "PARTIALLY_SUPPORTED") cls = "claim-partial";

    const claimSlice = answer.substring(c.start_offset, c.end_offset);
    html += `<span class="claim-highlight ${cls}" data-claim-id="${c.id}" title="[#${c.id} ${c.status}] ${escapeHTML(c.explanation)}">${escapeHTML(claimSlice)}</span>`;
    lastIdx = c.end_offset;
  }

  if (lastIdx < answer.length) {
    html += escapeHTML(answer.substring(lastIdx));
  }

  container.innerHTML = html;

  // Clicking highlight scrolls to table row
  container.querySelectorAll(".claim-highlight").forEach(el => {
    el.addEventListener("click", () => {
      const id = el.dataset.claimId;
      const row = document.getElementById(`claim-row-${id}`);
      if (row) {
        row.scrollIntoView({ behavior: "smooth", block: "center" });
        row.style.background = "rgba(59, 130, 246, 0.25)";
        setTimeout(() => { row.style.background = ""; }, 1500);
      }
    });
  });
}

function renderClaimsTable(claims) {
  const tbody = document.getElementById("claims-tbody");
  tbody.innerHTML = "";

  for (const c of claims) {
    const tr = document.createElement("tr");
    tr.id = `claim-row-${c.id}`;
    tr.dataset.status = c.status;

    let badgeClass = "badge-supported";
    if (c.status === "CONTRADICTED") badgeClass = "badge-contradicted";
    else if (c.status === "UNSUPPORTED") badgeClass = "badge-unsupported";
    else if (c.status === "PARTIALLY_SUPPORTED") badgeClass = "badge-partial";

    let evidenceHtml = `<span style="color: var(--text-muted); font-style: italic;">No grounding found</span>`;
    if (c.best_evidence) {
      evidenceHtml = `<div class="evidence-box">"${escapeHTML(c.best_evidence.snippet)}"</div>`;
    }

    let alertHtml = "";
    if (c.numeric_discrepancies && c.numeric_discrepancies.length > 0) {
      alertHtml += `<div style="color: #f87171; font-size: 0.8rem; margin-top: 4px;">⚠️ <strong>Numeric Divergence:</strong> ${escapeHTML(c.numeric_discrepancies[0].discrepancy)}</div>`;
    }
    if (c.missing_entities && c.missing_entities.length > 0) {
      alertHtml += `<div style="color: #fbbf24; font-size: 0.8rem; margin-top: 4px;">⚠️ <strong>Missing Entity:</strong> ${escapeHTML(c.missing_entities.join(", "))}</div>`;
    }

    tr.innerHTML = `
      <td style="font-weight: 700; color: var(--text-muted);">#${c.id}</td>
      <td style="max-width: 320px;">
        <div style="font-weight: 600;">${escapeHTML(c.text)}</div>
        <div style="font-size: 0.8rem; color: var(--text-muted); margin-top: 4px;">${escapeHTML(c.explanation)}</div>
        ${alertHtml}
      </td>
      <td><span class="badge-tag ${badgeClass}">${c.status}</span></td>
      <td><code>${c.hallucination_type}</code></td>
      <td style="font-weight: 600;">${Math.round(c.confidence * 100)}%</td>
      <td style="max-width: 380px;">${evidenceHtml}</td>
    `;

    tbody.appendChild(tr);
  }
}

function filterClaimsTable(status) {
  const rows = document.querySelectorAll("#claims-tbody tr");
  rows.forEach(row => {
    if (status === "ALL" || row.dataset.status === status) {
      row.style.display = "";
    } else {
      row.style.display = "none";
    }
  });
}

function exportJSON() {
  if (!currentResult) return;
  const blob = new Blob([JSON.stringify(currentResult, null, 2)], { type: "application/json" });
  downloadBlob(blob, `hallucination_report_${currentResult.id}.json`);
}

function exportMarkdown() {
  if (!currentResult) return;
  let md = `# 🛡️ LLM Hallucination Verification Report\n\n`;
  md += `**Verdict:** ${currentResult.verdict} | **Faithfulness:** ${Math.round(currentResult.faithfulness_score * 100)}%\n\n`;
  md += `### Claims Analysis\n`;
  for (const c of currentResult.claims) {
    md += `- **[${c.status}]** ${c.text}\n`;
    md += `  - Type: ${c.hallucination_type} (Confidence: ${Math.round(c.confidence * 100)}%)\n`;
    md += `  - Explanation: ${c.explanation}\n`;
    if (c.best_evidence) {
      md += `  - Grounding Snippet: "${c.best_evidence.snippet}"\n`;
    }
  }
  const blob = new Blob([md], { type: "text/markdown" });
  downloadBlob(blob, `hallucination_report_${currentResult.id}.md`);
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function escapeHTML(str) {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
