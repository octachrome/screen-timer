namespace ScreenTimer.Agent.Core.Models;

public sealed class GroupUsageState
{
    public bool Sent10Min { get; set; }
    public bool Sent5Min { get; set; }
    public bool Sent1Min { get; set; }
    public bool Exhausted { get; set; }
}
