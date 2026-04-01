namespace ScreenTimer.Agent.Core.Interfaces;

public interface INotificationSink
{
    void ShowToast(string exeName, int remainingMinutes);
}
