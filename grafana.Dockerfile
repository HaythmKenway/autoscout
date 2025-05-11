FROM grafana/grafana-enterprise:latest

# Switch to root for privileged operations
USER root

# Create required directories
RUN mkdir -p /etc/grafana/provisioning/dashboards /etc/grafana/dashboards

# Copy provisioning config and dashboard JSON
COPY provisioning/dashboards/my-dashboards.yml /etc/grafana/provisioning/dashboards/
COPY dashboards/autoscout.json /etc/grafana/dashboards/

# Set proper ownership

# Switch back to the Grafana user
USER grafana

EXPOSE 3000
