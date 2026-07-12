import os

class IpStatus:
    def __init__(self, ip: str, loss_rate: float, delay: float, download_speed: float):
        self.ip = ip
        self.loss_rate = loss_rate
        self.delay = delay
        self.download_speed = download_speed

    def __lt__(self, other):
        if abs(self.loss_rate - other.loss_rate) > 1e-5:
            return self.loss_rate < other.loss_rate
        if abs(self.download_speed - other.download_speed) > 1e-5:
            return self.download_speed > other.download_speed
        return self.delay < other.delay

    def __str__(self):
        return f"{self.ip},{self.loss_rate},{self.delay},{self.download_speed}"

    def __repr__(self):
        return self.__str__()
    

def get_all_result_files(directory):
    result_files = [os.path.join(directory, f) for f in os.listdir(directory) if f.endswith('.csv')]
    return result_files

def read_result_file(file_path):
    ip_status_list = []
    file = open(file_path, 'r', encoding='utf-8')
    lines = []
    try:
        lines = file.readlines()
    except Exception as e:
        print(f"Error reading file {file_path}: {e}")
    finally:
        file.close()
    for line in lines:
        line_split = line.split(',')
        if len(line_split) < 7:
            continue
        ip, _send, _recive, loss_rate, delay, download_speed, _location = line_split
        if ip == "IP 地址":
            continue
        ip_status_list.append(IpStatus(ip, float(loss_rate), float(delay), float(download_speed)))
    return ip_status_list

def gen_result_summary():
    result_files = get_all_result_files("./")
    ip_status_list = []
    for file in result_files:
        ip_status_list.extend(read_result_file(file))
    ip_status_list.sort()

    summary_file = open("result_summary.csv", 'w', encoding='utf-8')
    summary_file.write("IP 地址,丢包率,延迟,下载速度\n")
    for ip_status in ip_status_list:
        if ip_status.loss_rate > 1e-5 or ip_status.download_speed < 0.1:
            continue
        summary_file.write(f"{ip_status.ip},{ip_status.loss_rate},{ip_status.delay},{ip_status.download_speed}\n")
    summary_file.close()


if __name__ == "__main__":
    # result_files = get_all_result_files("./")
    # print(result_files)
    # ip_status_list = read_result_file("./results.csv")
    # print(ip_status_list)
    gen_result_summary()
