namespace ScreenTimer.Agent.Core.Models;

public sealed class AgentState
{
    public Dictionary<string, AppUsageState> Apps { get; set; } = new(StringComparer.OrdinalIgnoreCase);
    public Dictionary<string, GroupUsageState> GroupUsage { get; set; } = new(StringComparer.OrdinalIgnoreCase);
    public string? LastForegroundExe { get; set; }
    public DateTimeOffset LastTickTime { get; set; }
    public DateTimeOffset LastConfigPollTime { get; set; }
    public DateTimeOffset LastUsageFlushTime { get; set; }
    public string CurrentDate { get; set; } = "";
    public List<GroupRule> CurrentRules { get; set; } = new();
    public DateTimeOffset? LastTestPopupTime { get; set; }
}
