using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.AspNetCore.Routing;
using Microsoft.AspNetCore.Http;

namespace netcore_sessions
{
    public class Startup
    {
        public Startup(IConfiguration configuration)
        {
            Configuration = configuration;
        }

        public IConfiguration Configuration { get; }

        // This method gets called by the runtime. Use this method to add services to the container.
        public void ConfigureServices(IServiceCollection services)
        {
            services.AddRouting();

            // Adds a default in-memory implementation of IDistributedCache.
            services.AddDistributedMemoryCache();

            services.AddSession(options =>
            {
                options.Cookie.Name = ".cookiesession.id";
                options.Cookie.HttpOnly = true;
                options.IdleTimeout = TimeSpan.FromMinutes(1);
            });
        }

        // This method gets called by the runtime. Use this method to configure the HTTP request pipeline.
        public void Configure(IApplicationBuilder app, IHostingEnvironment env)
        {
            var routeBuilder = new RouteBuilder(app);

            routeBuilder.MapGet("setget", context =>{
                context.Session.SetString("key", "value");

                var value = context.Session.GetString("key");
                if (String.IsNullOrEmpty(value)) {
                    return context.Response.WriteAsync("NOT_OK");
                }

                return context.Response.WriteAsync(value);
            });
            /*
            Test them one by one by these methods:

            routeBuilder.MapGet("get", context =>{
                var value = context.Session.GetString("key");
                if (String.IsNullOrEmpty(value)) {
                    return context.Response.WriteAsync("NOT_OK");
                }
                return context.Response.WriteAsync(value);
            });

            routeBuilder.MapPost("set", context =>{
                context.Session.SetString("key", "value");
                return context.Response.WriteAsync("OK");
            });

            routeBuilder.MapDelete("del", context =>{
                context.Session.Remove("key");
                return context.Response.WriteAsync("OK");
            });
            */

            var routes = routeBuilder.Build();

            app.UseSession();
            app.UseRouter(routes);
        }
    }
}
