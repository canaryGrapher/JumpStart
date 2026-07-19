// Graphified from the real dashboard stat cards.
export default function WinStats() {
  return (
    <div class="win-layer win-stats" data-layer="stats">
      <div class="stat-card">
        <h5>Projects</h5>
        <div class="stat-line"><b>20</b><span class="stat-chip">0 running</span></div>
        <p class="stat-sub">subprocesses currently managed</p>
      </div>
      <div class="stat-card">
        <h5>System CPU</h5>
        <div class="stat-line"><b>14%</b><span class="stat-chip plain">12 cores</span></div>
        <div class="meter"><i style="width:14%"></i></div>
      </div>
      <div class="stat-card">
        <h5>System memory</h5>
        <div class="stat-line"><b>63%</b><span class="stat-chip plain">15/24 GB</span></div>
        <div class="meter"><i style="width:63%"></i></div>
      </div>
      <div class="stat-card">
        <h5>Features built</h5>
        <div class="stat-line"><b>192/304</b><span class="stat-chip">63%</span></div>
        <p class="stat-sub">across all task trackers</p>
        <div class="meter"><i style="width:63%"></i></div>
      </div>
    </div>
  );
}
