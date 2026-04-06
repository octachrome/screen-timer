namespace ScreenTimer.Agent.Core.Models;

public sealed class AppUsageState
{
    public double UsedTodaySeconds { get; set; }
    public double PendingUploadSeconds { get; set; }
}
