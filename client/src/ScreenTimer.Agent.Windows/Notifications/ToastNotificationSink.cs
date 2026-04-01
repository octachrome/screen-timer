using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Notifications;

public sealed class ToastNotificationSink : INotificationSink
{
    public void ShowToast(string exeName, int remainingMinutes)
    {
        Console.WriteLine($"[NOTIFICATION] {exeName}: {remainingMinutes} minute(s) remaining");
        // TODO: Replace with actual Windows toast notification (e.g., Microsoft.Toolkit.Uwp.Notifications)
    }
}
