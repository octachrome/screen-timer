using System.Text.Json.Serialization;

namespace ScreenTimer.Agent.Core.Dtos;

public sealed class GroupConfigDto
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = "";

    [JsonPropertyName("processes")]
    public List<string> Processes { get; set; } = new();

    [JsonPropertyName("daily_budget_minutes")]
    public int DailyBudgetMinutes { get; set; }

    [JsonPropertyName("weekend_budget_minutes")]
    public int WeekendBudgetMinutes { get; set; }
}
