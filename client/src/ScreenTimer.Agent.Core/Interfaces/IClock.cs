namespace ScreenTimer.Agent.Core.Interfaces;

public interface IClock
{
    DateTimeOffset Now { get; }
}
