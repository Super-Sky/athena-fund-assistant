import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type RiskPreference = "conservative" | "balanced" | "aggressive";
type IconName = "activity" | "alert" | "bar" | "check" | "database" | "note" | "play" | "shield" | "wallet";

type InvestorProfile = {
  risk_preference: RiskPreference;
  investment_horizon_months: number;
  max_acceptable_drawdown_pct: number;
  single_instrument_max_allocation_pct: number;
  cash_preference_pct: number;
  default_decision_style: string;
};

type PortfolioHolding = {
  instrument_code: string;
  instrument_name: string;
  market: string;
  currency: string;
  holding_amount: number;
  cost_basis: number;
  allocation_pct: number;
  user_thesis: string;
};

type SourceMetadata = {
  source: string;
  provider: string;
  fetched_at: string;
  market_time: string;
  timezone: string;
  delay: string;
  license_terms: string;
  confidence: number;
  schema_version: string;
  raw_payload_hash?: string;
};

type FundSnapshot = {
  instrument: {
    code: string;
    name: string;
    market: string;
    currency: string;
    type: string;
  };
  nav?: number;
  price?: number;
  daily_change_pct: number;
  one_year_return_pct: number;
  max_drawdown_pct: number;
  volatility_pct: number;
  expense_ratio_pct?: number;
  manager?: string;
  asset_size?: string;
  top_holdings?: string[];
  metadata: SourceMetadata;
};

type Diagnosis = {
  instrument_code: string;
  summary: string;
  risk_factors: string[] | null;
  data_warnings: string[] | null;
  evidence: string[] | null;
};

type DecisionOption = {
  id: string;
  style: string;
  action: string;
  allocation_change_pct: number;
  conditions: string[] | null;
  evidence: string[] | null;
  risks: string[] | null;
  invalidation: string;
  review_after_days: number;
  portfolio_impact: string;
  strategy_basis: string[] | null;
};

type TraceSummary = {
  data_provider: string;
  data_source: string;
  data_fetched_at: string;
  market_time: string;
  timezone: string;
  license_terms: string;
  confidence: number;
  rule_evaluations: string[] | null;
  governance_checks: string[] | null;
  athena_run_id?: string;
  mock_data_temporary: boolean;
};

type DecisionMatrix = {
  id: string;
  instrument: FundSnapshot["instrument"];
  generated_at: string;
  options: DecisionOption[];
  governance_tags: string[] | null;
  trace: TraceSummary;
};

type AnalysisResponse = {
  profile: InvestorProfile;
  portfolio: {
    holdings: PortfolioHolding[];
  };
  fund_snapshot: FundSnapshot;
  diagnosis: Diagnosis;
  decision_matrix: DecisionMatrix;
};

type JournalResponse = {
  journal: {
    id: string;
    created_at: string;
    selected_option_id: string;
  };
  review: {
    id: string;
    due_at: string;
    question: string;
    trigger_hint: string;
    status: string;
  };
};

type AccountHoldingSnapshot = {
  id: string;
  instrument_code: string;
  instrument_name: string;
  market: string;
  currency: string;
  units: number;
  cost_basis: number;
  current_price: number;
  fx_to_base: number;
  base_market_value: number;
  base_cost_value: number;
  unrealized_pnl: number;
  unrealized_pnl_pct: number;
  allocation_pct: number;
  data_authorization: string;
  metadata: SourceMetadata;
};

type AccountPerformancePoint = {
  date: string;
  total_market_value: number;
  total_cost_value: number;
  total_pnl: number;
  total_pnl_pct: number;
  operation_pnl: number;
};

type AccountOverview = {
  account: {
    user_id: string;
    display_name: string;
    base_currency: string;
    auth_mode: string;
  };
  holdings: AccountHoldingSnapshot[];
  total_market_value: number;
  total_cost_value: number;
  total_pnl: number;
  total_pnl_pct: number;
  recent_operation_pnl: number;
  base_currency: string;
  performance_trend: AccountPerformancePoint[];
  trace: {
    provider: string;
    source: string;
    market_time: string;
    timezone: string;
    mock_data_temporary: boolean;
    read_only_sync_available: boolean;
    warnings: string[] | null;
  };
};

type InstrumentPreset = {
  code: string;
  name: string;
  market: string;
  currency: string;
  amount: number;
  cost: number;
  allocation: number;
  thesis: string;
};

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "";

const presets: InstrumentPreset[] = [
  {
    code: "510300",
    name: "Sample CSI 300 ETF",
    market: "CN",
    currency: "CNY",
    amount: 50000,
    cost: 4.2,
    allocation: 22,
    thesis: "宽基权益 beta，关注回撤与单标的集中度"
  },
  {
    code: "QQQ",
    name: "Sample Nasdaq 100 ETF",
    market: "US",
    currency: "USD",
    amount: 12000,
    cost: 420,
    allocation: 18,
    thesis: "美股科技成长敞口，接受波动但不突破仓位上限"
  },
  {
    code: "000001",
    name: "Sample China Balanced Fund",
    market: "CN",
    currency: "CNY",
    amount: 30000,
    cost: 1.1,
    allocation: 12,
    thesis: "偏稳健配置，用于观察基金经理风格与净值回撤"
  }
];

const defaultProfile: InvestorProfile = {
  risk_preference: "balanced",
  investment_horizon_months: 24,
  max_acceptable_drawdown_pct: 25,
  single_instrument_max_allocation_pct: 20,
  cash_preference_pct: 8,
  default_decision_style: "three_options"
};

function holdingFromPreset(preset: InstrumentPreset): PortfolioHolding {
  return {
    instrument_code: preset.code,
    instrument_name: preset.name,
    market: preset.market,
    currency: preset.currency,
    holding_amount: preset.amount,
    cost_basis: preset.cost,
    allocation_pct: preset.allocation,
    user_thesis: preset.thesis
  };
}

function App() {
  const [instrumentCode, setInstrumentCode] = useState(presets[0].code);
  const [profile, setProfile] = useState<InvestorProfile>(defaultProfile);
  const [holding, setHolding] = useState<PortfolioHolding>(holdingFromPreset(presets[0]));
  const [analysis, setAnalysis] = useState<AnalysisResponse | null>(null);
  const [selectedOptionID, setSelectedOptionID] = useState("option_balanced");
  const [notes, setNotes] = useState("按本轮策略执行前，先确认数据源仍为 mock；真实资金操作需自行复核。");
  const [journal, setJournal] = useState<JournalResponse | null>(null);
  const [accountOverview, setAccountOverview] = useState<AccountOverview | null>(null);
  const [loadingAnalysis, setLoadingAnalysis] = useState(false);
  const [loadingAccount, setLoadingAccount] = useState(false);
  const [savingJournal, setSavingJournal] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedOption = useMemo(
    () => analysis?.decision_matrix.options.find((option) => option.id === selectedOptionID) ?? null,
    [analysis, selectedOptionID]
  );

  useEffect(() => {
    let cancelled = false;
    setLoadingAccount(true);
    fetchJSON<AccountOverview>("/api/accounts/demo-user/overview", { method: "GET" })
      .then((response) => {
        if (!cancelled) {
          setAccountOverview(response);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingAccount(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  function applyPreset(code: string) {
    const preset = presets.find((item) => item.code === code) ?? presets[0];
    setInstrumentCode(preset.code);
    setHolding(holdingFromPreset(preset));
    setAnalysis(null);
    setJournal(null);
    setSelectedOptionID("option_balanced");
  }

  async function runAnalysis(event: React.FormEvent) {
    event.preventDefault();
    setLoadingAnalysis(true);
    setError(null);
    setJournal(null);
    try {
      const response = await fetchJSON<AnalysisResponse>("/api/analysis/fund", {
        method: "POST",
        body: JSON.stringify({
          instrument_code: instrumentCode,
          profile,
          portfolio: {
            holdings: [holding]
          }
        })
      });
      setAnalysis(response);
      const balanced = response.decision_matrix.options.find((option) => option.id === "option_balanced");
      setSelectedOptionID(balanced?.id ?? response.decision_matrix.options[0]?.id ?? "");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingAnalysis(false);
    }
  }

  async function saveJournal() {
    if (!analysis || !selectedOptionID) {
      return;
    }
    setSavingJournal(true);
    setError(null);
    try {
      const response = await fetchJSON<JournalResponse>("/api/journals", {
        method: "POST",
        body: JSON.stringify({
          matrix: analysis.decision_matrix,
          selected_option_id: selectedOptionID,
          user_notes: notes
        })
      });
      setJournal(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSavingJournal(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Athena Fund Assistant</p>
          <h1>基金研究决策台</h1>
        </div>
        <div className="status-strip" aria-label="runtime status">
          <StatusPill icon="shield" label="先验证后编码" tone="green" />
          <StatusPill icon="database" label="Mock 数据" tone="amber" />
          <StatusPill icon="check" label="无自动交易" tone="blue" />
        </div>
      </header>

      <AccountDashboard overview={accountOverview} loading={loadingAccount} />

      <section className="workspace">
        <form className="control-panel" onSubmit={runAnalysis}>
          <PanelTitle icon="wallet" title="输入" caption="用户画像与单标的持仓" />

          <div className="preset-row" role="group" aria-label="instrument presets">
            {presets.map((preset) => (
              <button
                className={preset.code === instrumentCode ? "chip active" : "chip"}
                key={preset.code}
                onClick={() => applyPreset(preset.code)}
                type="button"
              >
                {preset.code}
              </button>
            ))}
          </div>

          <label className="field">
            <span>风险偏好</span>
            <select
              value={profile.risk_preference}
              onChange={(event) =>
                setProfile({ ...profile, risk_preference: event.target.value as RiskPreference })
              }
            >
              <option value="conservative">稳健</option>
              <option value="balanced">均衡</option>
              <option value="aggressive">激进</option>
            </select>
          </label>

          <div className="field-grid">
            <NumberField
              label="投资周期(月)"
              value={profile.investment_horizon_months}
              onChange={(value) => setProfile({ ...profile, investment_horizon_months: value })}
            />
            <NumberField
              label="最大回撤(%)"
              value={profile.max_acceptable_drawdown_pct}
              onChange={(value) => setProfile({ ...profile, max_acceptable_drawdown_pct: value })}
            />
            <NumberField
              label="单标的上限(%)"
              value={profile.single_instrument_max_allocation_pct}
              onChange={(value) => setProfile({ ...profile, single_instrument_max_allocation_pct: value })}
            />
            <NumberField
              label="现金偏好(%)"
              value={profile.cash_preference_pct}
              onChange={(value) => setProfile({ ...profile, cash_preference_pct: value })}
            />
          </div>

          <div className="divider" />

          <label className="field">
            <span>标的代码</span>
            <input value={instrumentCode} onChange={(event) => setInstrumentCode(event.target.value)} />
          </label>

          <label className="field">
            <span>标的名称</span>
            <input
              value={holding.instrument_name}
              onChange={(event) => setHolding({ ...holding, instrument_name: event.target.value })}
            />
          </label>

          <div className="field-grid">
            <label className="field">
              <span>市场</span>
              <input value={holding.market} onChange={(event) => setHolding({ ...holding, market: event.target.value })} />
            </label>
            <label className="field">
              <span>币种</span>
              <input
                value={holding.currency}
                onChange={(event) => setHolding({ ...holding, currency: event.target.value })}
              />
            </label>
            <NumberField
              label="持有金额"
              value={holding.holding_amount}
              onChange={(value) => setHolding({ ...holding, holding_amount: value })}
            />
            <NumberField
              label="当前占比(%)"
              value={holding.allocation_pct}
              onChange={(value) => setHolding({ ...holding, allocation_pct: value })}
            />
          </div>

          <label className="field">
            <span>持有理由</span>
            <textarea
              value={holding.user_thesis}
              onChange={(event) => setHolding({ ...holding, user_thesis: event.target.value })}
            />
          </label>

          <button className="primary-action" disabled={loadingAnalysis} type="submit">
            <Icon name="play" />
            {loadingAnalysis ? "分析中" : "生成三档策略"}
          </button>
        </form>

        <section className="output-panel">
          {error ? (
            <div className="error-banner" role="alert">
              <Icon name="alert" />
              {error}
            </div>
          ) : null}

          {analysis ? (
            <>
              <SnapshotView analysis={analysis} />
              <DecisionOptions
                options={analysis.decision_matrix.options}
                selectedOptionID={selectedOptionID}
                onSelect={setSelectedOptionID}
              />
              <JournalBox
                disabled={!selectedOption}
                journal={journal}
                notes={notes}
                saving={savingJournal}
                selectedOption={selectedOption}
                onNotesChange={setNotes}
                onSave={saveJournal}
              />
            </>
          ) : (
            <EmptyState />
          )}
        </section>
      </section>
    </main>
  );
}

function AccountDashboard({ loading, overview }: { loading: boolean; overview: AccountOverview | null }) {
  if (loading && !overview) {
    return (
      <section className="account-dashboard">
        <PanelTitle icon="wallet" title="账户收益" caption="正在读取本地账户快照" />
      </section>
    );
  }
  if (!overview) {
    return null;
  }
  const lastPoint = overview.performance_trend[overview.performance_trend.length - 1];
  return (
    <section className="account-dashboard">
      <div className="account-head">
        <PanelTitle
          icon="wallet"
          title={`账户收益 · ${overview.account.display_name} · ${overview.base_currency}`}
          caption="账户总收益、近期操作收益与持仓结构"
        />
        <div className="account-flags">
          <span className="tag evidence">{overview.trace.provider}</span>
          <span className="tag warning">{overview.trace.mock_data_temporary ? "mock temporary" : "live data"}</span>
          <span className="tag evidence">
            {overview.trace.read_only_sync_available ? "read-only sync ready" : "manual entry"}
          </span>
        </div>
      </div>

      <div className="account-metrics">
        <Metric label="总市值" value={formatCurrency(overview.total_market_value, overview.base_currency)} />
        <Metric
          label="总收益"
          tone={overview.total_pnl >= 0 ? "up" : "down"}
          value={`${formatCurrency(overview.total_pnl, overview.base_currency)} · ${signedPct(overview.total_pnl_pct)}`}
        />
        <Metric
          label="近期操作收益"
          tone={overview.recent_operation_pnl >= 0 ? "up" : "down"}
          value={formatCurrency(overview.recent_operation_pnl, overview.base_currency)}
        />
        <Metric label="最近趋势" value={lastPoint ? signedPct(lastPoint.total_pnl_pct) : "-"} />
      </div>

      <div className="account-body">
        <div className="holding-list">
          {overview.holdings.map((holding) => (
            <div className="holding-row" key={holding.id}>
              <div>
                <strong>{holding.instrument_code}</strong>
                <span>{holding.instrument_name}</span>
              </div>
              <div>
                <strong>{formatCurrency(holding.base_market_value, overview.base_currency)}</strong>
                <span>
                  {holding.market} · {holding.currency} · {formatPct(holding.allocation_pct)}
                </span>
              </div>
              <div>
                <strong className={holding.unrealized_pnl >= 0 ? "positive" : "negative"}>
                  {signedPct(holding.unrealized_pnl_pct)}
                </strong>
                <span>{holding.data_authorization}</span>
              </div>
            </div>
          ))}
        </div>
        <div className="trend-strip" aria-label="account performance trend">
          {overview.performance_trend.map((point) => (
            <div className="trend-point" key={point.date}>
              <span>{point.date.slice(5)}</span>
              <strong className={point.total_pnl >= 0 ? "positive" : "negative"}>{signedPct(point.total_pnl_pct)}</strong>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function PanelTitle({ caption, icon, title }: { caption: string; icon: IconName; title: string }) {
  return (
    <div className="panel-title">
      <div className="panel-title-icon">
        <Icon name={icon} />
      </div>
      <div>
        <h2>{title}</h2>
        <p>{caption}</p>
      </div>
    </div>
  );
}

function StatusPill({ icon, label, tone }: { icon: IconName; label: string; tone: string }) {
  return (
    <span className={`status-pill ${tone}`}>
      <Icon name={icon} />
      {label}
    </span>
  );
}

function Icon({ name }: { name: IconName }) {
  return <span aria-hidden="true" className={`ui-icon ${name}`} />;
}

function NumberField({
  label,
  onChange,
  value
}: {
  label: string;
  onChange: (value: number) => void;
  value: number;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      <input inputMode="decimal" type="number" value={value} onChange={(event) => onChange(Number(event.target.value))} />
    </label>
  );
}

function SnapshotView({ analysis }: { analysis: AnalysisResponse }) {
  const snapshot = analysis.fund_snapshot;
  const trace = analysis.decision_matrix.trace;
  return (
    <section className="snapshot-grid">
      <div className="metric-board">
        <PanelTitle icon="bar" title={snapshot.instrument.code} caption={snapshot.instrument.name} />
        <div className="metric-row">
          <Metric label="价格/NAV" value={formatNumber(snapshot.price ?? snapshot.nav ?? 0)} />
          <Metric label="日涨跌" value={signedPct(snapshot.daily_change_pct)} tone={snapshot.daily_change_pct >= 0 ? "up" : "down"} />
          <Metric label="一年收益" value={signedPct(snapshot.one_year_return_pct)} tone={snapshot.one_year_return_pct >= 0 ? "up" : "down"} />
          <Metric label="最大回撤" value={formatPct(snapshot.max_drawdown_pct)} tone="risk" />
          <Metric label="波动率" value={formatPct(snapshot.volatility_pct)} />
        </div>
        <p className="diagnosis">{analysis.diagnosis.summary}</p>
        <TagList items={analysis.diagnosis.risk_factors ?? []} empty="当前画像约束内未触发额外风险" tone="risk" />
        <TagList items={analysis.diagnosis.data_warnings ?? []} empty="数据告警为空" tone="warning" />
      </div>

      <div className="trace-board">
        <PanelTitle icon="activity" title="Trace" caption="数据来源与治理状态" />
        <dl className="trace-list">
          <TraceItem label="provider" value={trace.data_provider} />
          <TraceItem label="source" value={trace.data_source} />
          <TraceItem label="license" value={trace.license_terms} />
          <TraceItem label="confidence" value={`${Math.round(trace.confidence * 100)}%`} />
          <TraceItem label="market time" value={formatDate(trace.market_time)} />
          <TraceItem label="fetched at" value={formatDate(trace.data_fetched_at)} />
          <TraceItem label="timezone" value={trace.timezone} />
          <TraceItem label="mock" value={trace.mock_data_temporary ? "temporary" : "false"} />
        </dl>
      </div>
    </section>
  );
}

function DecisionOptions({
  onSelect,
  options,
  selectedOptionID
}: {
  onSelect: (id: string) => void;
  options: DecisionOption[];
  selectedOptionID: string;
}) {
  return (
    <section className="options-panel">
      <PanelTitle icon="shield" title="三档策略" caption="同一事实下的稳健、均衡、激进路径" />
      <div className="option-grid">
        {options.map((option) => (
          <button
            className={option.id === selectedOptionID ? "option-card selected" : "option-card"}
            key={option.id}
            onClick={() => onSelect(option.id)}
            type="button"
          >
            <span className="option-topline">
              <span>{styleLabel(option.style)}</span>
              <strong>{signedPct(option.allocation_change_pct)}</strong>
            </span>
            <span className="option-action">{option.action}</span>
            <span className="option-impact">{option.portfolio_impact}</span>
            <span className="option-meta">复盘：{option.review_after_days} 天</span>
          </button>
        ))}
      </div>
    </section>
  );
}

function JournalBox({
  disabled,
  journal,
  notes,
  onNotesChange,
  onSave,
  saving,
  selectedOption
}: {
  disabled: boolean;
  journal: JournalResponse | null;
  notes: string;
  onNotesChange: (value: string) => void;
  onSave: () => void;
  saving: boolean;
  selectedOption: DecisionOption | null;
}) {
  return (
    <section className="journal-panel">
      <PanelTitle icon="note" title="Decision Journal" caption="记录选择与复盘触发条件" />
      {selectedOption ? (
        <div className="option-detail">
          <p>{selectedOption.invalidation}</p>
          <TagList items={selectedOption.evidence ?? []} empty="暂无依据" tone="evidence" />
          <TagList items={selectedOption.risks ?? []} empty="暂无风险项" tone="risk" />
        </div>
      ) : null}
      <label className="field">
        <span>备注</span>
        <textarea value={notes} onChange={(event) => onNotesChange(event.target.value)} />
      </label>
      <button className="secondary-action" disabled={disabled || saving} onClick={onSave} type="button">
        <Icon name="note" />
        {saving ? "保存中" : "保存 journal"}
      </button>
      {journal ? (
        <div className="review-box">
          <strong>{journal.review.status.toUpperCase()}</strong>
          <span>{journal.review.question}</span>
          <small>Due: {formatDate(journal.review.due_at)}</small>
        </div>
      ) : null}
    </section>
  );
}

function EmptyState() {
  return (
    <section className="empty-state">
      <div className="empty-icon">
        <Icon name="activity" />
      </div>
      <h2>等待生成策略</h2>
      <p>选择标的并提交后，这里会显示基金体检、三档策略、trace 和 journal 复盘任务。</p>
    </section>
  );
}

function Metric({ label, tone, value }: { label: string; tone?: string; value: string }) {
  return (
    <div className={`metric ${tone ?? ""}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function TraceItem({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </>
  );
}

function TagList({ empty, items, tone }: { empty: string; items: string[]; tone: string }) {
  const values = items.length > 0 ? items : [empty];
  return (
    <div className="tag-list">
      {values.map((item) => (
        <span className={`tag ${tone}`} key={item}>
          {item}
        </span>
      ))}
    </div>
  );
}

async function fetchJSON<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {})
    }
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.error ?? `request failed with ${response.status}`);
  }
  return payload as T;
}

function styleLabel(style: string) {
  if (style === "conservative") {
    return "稳健";
  }
  if (style === "aggressive") {
    return "激进";
  }
  return "均衡";
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(value);
}

function formatCurrency(value: number, currency: string) {
  return new Intl.NumberFormat("zh-CN", {
    currency,
    maximumFractionDigits: 2,
    style: "currency"
  }).format(value);
}

function formatPct(value: number) {
  return `${formatNumber(value)}%`;
}

function signedPct(value: number) {
  return `${value > 0 ? "+" : ""}${formatPct(value)}`;
}

function formatDate(value: string) {
  if (!value) {
    return "-";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
