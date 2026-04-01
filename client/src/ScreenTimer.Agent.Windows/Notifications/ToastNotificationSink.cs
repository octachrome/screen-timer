using Microsoft.Toolkit.Uwp.Notifications;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Notifications;

public sealed class ToastNotificationSink : INotificationSink
{
    public void ShowToast(string exeName, int remainingMinutes)
    {
        new ToastContentBuilder()
            .AddText("Screen Timer")
            .AddText($"{exeName}: {remainingMinutes} minute(s) remaining")
            .Show(toast =>
            {
                toast.Tag = $"screentimer-{exeName}";
                toast.ExpirationTime = DateTimeOffset.Now.AddMinutes(Math.Max(remainingMinutes, 1));
            });
    }
}
