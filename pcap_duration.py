from scapy.all import rdpcap
packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
start = packets[0].time
end = packets[-1].time
print(f"Pcap duration: {end - start} seconds")
