namespace ScreenTimer.Agent.Core.Models;

public sealed class GroupRule
{
    public required string Name { get; init; }
    public List<string> Processes { get; set; } = new();
    public int DailyBudgetMinutes { get; set; }
    public int WeekendBudgetMinutes { get; set; }
}
