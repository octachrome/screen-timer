using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Host;
using ScreenTimer.Agent.Windows;
using ScreenTimer.Agent.Windows.Foreground;
using ScreenTimer.Agent.Windows.Http;
using ScreenTimer.Agent.Windows.Notifications;
using ScreenTimer.Agent.Windows.Processes;
using ScreenTimer.Agent.Windows.Storage;

var builder = Host.CreateApplicationBuilder(args);

var serverUrl = builder.Configuration.GetValue<string>("ServerUrl") ?? "http://localhost:8080";

builder.Services.AddSingleton<IClock, SystemClock>();
builder.Services.AddSingleton<IForegroundWindowProbe, Win32ForegroundWindowProbe>();
builder.Services.AddSingleton<INotificationSink, ToastNotificationSink>();
builder.Services.AddSingleton<IProcessController, WindowsProcessController>();
builder.Services.AddSingleton<IStateStore>(_ => new JsonStateStore(JsonStateStore.DefaultFilePath));
builder.Services.AddHttpClient<IAgentApiClient, AgentApiClient>(client =>
{
    client.BaseAddress = new Uri(serverUrl);
});
builder.Services.AddHostedService<AgentWorker>();

var host = builder.Build();
host.Run();
