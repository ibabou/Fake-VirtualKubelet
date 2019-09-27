FROM gcr.io/distroless/base

ENV KUBELET_PORT 10250

# Use the pre-built binary in "bin/virtual-kubelet".
COPY fake-vn /virtual-kubelet
# Copy the configuration file for the provider.
COPY vkubelet-cfg.json /vkubelet-cfg.json

CMD ["/virtual-kubelet"]
