using Microsoft.Toolkit.Uwp.Notifications;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Notifications;

public sealed class ToastNotificationSink : INotificationSink
{
    public void ShowToast(string label, int remainingMinutes)
    {
        new ToastContentBuilder()
            .AddText("Screen Timer")
            .AddText($"{remainingMinutes} minute(s) remaining for {label}")
            .Show(toast =>
            {
                toast.Tag = $"screentimer-{label}";
                toast.ExpirationTime = DateTimeOffset.Now.AddMinutes(Math.Max(remainingMinutes, 1));
            });
    }
}
