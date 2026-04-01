using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows;

public sealed class SystemClock : IClock
{
    public DateTimeOffset Now => DateTimeOffset.Now;
}
