namespace ScreenTimer.Agent.Core.Interfaces;

public interface IProcessController
{
    Task ForceCloseAsync(string exeName);
}
