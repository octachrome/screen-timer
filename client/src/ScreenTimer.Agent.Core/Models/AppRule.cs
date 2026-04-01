namespace ScreenTimer.Agent.Core.Models;

public sealed class AppRule
{
    public required string ExeName { get; init; }
    public int DailyBudgetMinutes { get; set; }
}
