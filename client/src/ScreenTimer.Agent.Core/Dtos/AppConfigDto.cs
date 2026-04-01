using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class AppConfigDto
{
    [JsonPropertyName("exe_name")]
    public string ExeName { get; set; } = "";

    [JsonPropertyName("daily_budget_minutes")]
    public int DailyBudgetMinutes { get; set; }
}
