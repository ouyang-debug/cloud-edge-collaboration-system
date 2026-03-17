import paramiko
import json


def ssh_connect(hostname, port, username, password):
    """
    SSH连接到服务器
    :param hostname: 服务器主机名或IP地址
    :param port: SSH端口，默认为22
    :param username: 用户名
    :param password: 密码
    :return: SSH客户端对象
    """
    try:
        # 创建SSH客户端
        client = paramiko.SSHClient()
        # 自动添加主机密钥
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        # 连接服务器
        client.connect(hostname, port, username, password)
        print(f"✅ 成功连接到服务器 {hostname}:{port}")
        return client
    except Exception as e:
        print(f"❌ 连接服务器失败: {str(e)}")
        return None


def execute_command(client, command):
    """
    执行SSH命令
    :param client: SSH客户端对象
    :param command: 要执行的命令
    :return: 命令执行结果
    """
    try:
        # 执行命令
        stdin, stdout, stderr = client.exec_command(command)
        # 获取命令输出
        output = stdout.read().decode('utf-8')
        error = stderr.read().decode('utf-8')

        if error:
            print(f"⚠️ 命令执行警告: {error}")

        return output
    except Exception as e:
        print(f"❌ 执行命令失败: {str(e)}")
        return None


def view_file(client, file_path):
    """
    查看文件内容
    :param client: SSH客户端对象
    :param file_path: 文件路径
    :return: 文件内容
    """
    try:
        # 执行cat命令查看文件内容
        command = f"cat {file_path}"
        content = execute_command(client, command)
        return content
    except Exception as e:
        print(f"❌ 查看文件失败: {str(e)}")
        return None


def pretty_print_json(content):
    """
    美化打印JSON内容
    :param content: JSON字符串
    :return: 美化后的JSON字符串
    """
    try:
        # 尝试解析JSON
        json_obj = json.loads(content)
        # 美化输出
        return json.dumps(json_obj, indent=2, ensure_ascii=False)
    except json.JSONDecodeError:
        # 如果不是JSON，返回原始内容
        return content


def get_latest_folder(client, directory):
    """
    获取目录下最新日期的文件夹
    :param client: SSH客户端对象
    :param directory: 目录路径
    :return: 最新文件夹名称
    """
    try:
        # 执行ls -la命令查看目录内容
        ls_output = execute_command(client, f"ls -la {directory}")
        if not ls_output:
            return None

        # 解析ls -la输出，提取文件夹名称和修改日期
        lines = ls_output.strip().split('\n')
        folders = []

        for line in lines:
            # 跳过第一行（总用量）和当前目录（.）、上级目录（..）
            if line.startswith('总用量') or line.endswith('.') or line.endswith('..'):
                continue
            # 提取修改日期和文件夹名称
            parts = line.split()
            if len(parts) >= 8:
                # 提取日期（第6-8列）
                date_str = ' '.join(parts[5:8])
                # 提取文件夹名称（最后一列）
                folder_name = parts[-1]
                # 检查是否是文件夹（以d开头）
                if line.startswith('d'):
                    folders.append((date_str, folder_name))

        # 找到最新日期的文件夹
        if folders:
            # 简单排序，假设日期格式为 "月 日 时间" 或 "月 日 年"
            folders.sort(reverse=True, key=lambda x: x[0])
            return folders[0][1]
        else:
            return None
    except Exception as e:
        print(f"❌ 获取最新文件夹失败: {str(e)}")
        return None


def main():
    # 服务器连接信息
    servers = [
        {"ip": "10.220.42.153", "port": 22},
        {"ip": "10.220.42.154", "port": 8090},
        {"ip": "10.220.42.155", "port": 22}
    ]
    username = "root"  # 用户名
    password = "Tpri@hn20251205"  # 密码

    # 服务器选择菜单
    print("=== 服务器选择 ===")
    for i, server in enumerate(servers, 1):
        print(f"{i}. {server['ip']}:{server['port']}")
    
    while True:
        choice = input("请选择要连接的服务器编号: ")
        if choice.isdigit() and 1 <= int(choice) <= len(servers):
            selected_server = servers[int(choice) - 1]
            hostname = selected_server['ip']
            port = selected_server['port']
            print(f"\n选择连接服务器: {hostname}:{port}")
            break
        else:
            print("❌ 无效的选择，请重新输入")

    # 连接服务器
    client = ssh_connect(hostname, port, username, password)

    if client:
        try:
            while True:
                print("\n=== 操作菜单 ====")
                print("1. 查看/data/program/ceam-agent/target/agent/data/tasks.json文件内容")
                print("2. 查看/data/program/ceam-agent/target/agent/tasks目录并分析最新文件夹")
                print("3. 执行自定义文件路径替换")
                print("4. 查看原始文件784367510900805/out/seq1/result.json")
                print("5. 断开连接并选择其他服务器")
                print("6. 退出")

                choice = input("\n请选择操作编号: ")

                if choice == '1':
                    # 1. 查看/data/program/ceam-agent/target/agent/data/tasks.json文件内容
                    print("\n=== 查看/data/program/ceam-agent/target/agent/data/tasks.json文件内容 ===")
                    tasks_json_path = "/data/program/ceam-agent/target/agent/data/tasks.json"
                    # 首先执行ls -la命令查看文件信息
                    tasks_json_ls = execute_command(client, f"ls -la {tasks_json_path}")
                    print(tasks_json_ls)
                    # 然后查看文件内容
                    tasks_json_content = view_file(client, tasks_json_path)
                    if tasks_json_content:
                        # 美化打印JSON内容
                        pretty_content = pretty_print_json(tasks_json_content)
                        print(pretty_content)
                    else:
                        print("❌ 无法查看tasks.json文件内容")

                elif choice == '2':
                    # 2. 查看/data/program/ceam-agent/target/agent/tasks目录并分析最新文件夹
                    print("\n=== 执行ls -la查看/data/program/ceam-agent/target/agent/tasks目录 ===")
                    tasks_dir_path = "/data/program/ceam-agent/target/agent/tasks"
                    tasks_dir_ls = execute_command(client, f"ls -la {tasks_dir_path}")
                    print(tasks_dir_ls)

                    # 分析目录内容，找到最新日期的文件夹
                    print("\n=== 分析目录内容，找到最新日期的文件夹 ===")
                    latest_folder = get_latest_folder(client, tasks_dir_path)
                    if latest_folder:
                        print(f"最新日期的文件夹: {latest_folder}")

                        # 构造完整的文件路径
                        latest_file_path = f"/data/program/ceam-agent/target/agent/tasks/{latest_folder}/out/seq1/result.json"
                        print(f"最新文件路径: {latest_file_path}")

                        # 询问是否查看最新文件夹中的result.json文件
                        view_latest = input("\n是否查看最新文件夹中的result.json文件？(y/n): ").lower()
                        if view_latest == 'y':
                            print("\n=== 查看最新文件夹中的result.json文件 ===")
                            # 首先执行ls -la命令查看文件信息
                            latest_file_ls = execute_command(client, f"ls -la {latest_file_path}")
                            print(latest_file_ls)
                            # 然后查看文件内容
                            latest_file_content = view_file(client, latest_file_path)
                            if latest_file_content:
                                # 美化打印JSON内容
                                pretty_content = pretty_print_json(latest_file_content)
                                print(pretty_content)
                            else:
                                print("❌ 无法查看最新文件夹中的result.json文件内容")
                    else:
                        print("❌ 未找到文件夹")

                elif choice == '3':
                    # 3. 执行自定义文件路径替换
                    # 询问替换内容
                    replacement = input("请输入替换内容: ")
                    # 构造原始路径和替换后的路径
                    original_path = "/data/program/ceam-agent/target/agent/tasks/&&/out/seq1/result.json"
                    new_path = original_path.replace("&&", replacement)
                    print(f"\n原始路径: {original_path}")
                    print(f"替换后路径: {new_path}")

                    # 查看替换后的文件
                    print("\n=== 查看替换后的文件 ===")
                    # 首先执行ls -la命令查看文件信息
                    new_path_ls = execute_command(client, f"ls -la {new_path}")
                    print(new_path_ls)
                    # 然后查看文件内容
                    new_path_content = view_file(client, new_path)
                    if new_path_content:
                        # 美化打印JSON内容
                        pretty_content = pretty_print_json(new_path_content)
                        print(pretty_content)
                    else:
                        print("❌ 无法查看替换后的文件内容")

                elif choice == '4':
                    # 4. 查看原始文件784367510900805/out/seq1/result.json
                    file_path = "784367510900805/out/seq1/result.json"
                    print("\n=== 执行ls -la命令查看文件信息 ===")
                    ls_output = execute_command(client, f"ls -la {file_path}")
                    print(ls_output)

                    print("\n=== 查看文件内容 ===")
                    file_content = view_file(client, file_path)
                    if file_content:
                        # 美化打印JSON内容
                        pretty_content = pretty_print_json(file_content)
                        print(pretty_content)
                    else:
                        print("❌ 无法查看文件内容")

                elif choice == '5':
                    # 5. 断开连接并选择其他服务器
                    print("\n断开连接并选择其他服务器")
                    client.close()
                    print("✅ 已关闭SSH连接")
                    # 重新显示服务器选择菜单
                    print("\n=== 服务器选择 ===")
                    for i, server in enumerate(servers, 1):
                        print(f"{i}. {server['ip']}:{server['port']}")
                    
                    while True:
                        server_choice = input("请选择要连接的服务器编号: ")
                        if server_choice.isdigit() and 1 <= int(server_choice) <= len(servers):
                            selected_server = servers[int(server_choice) - 1]
                            hostname = selected_server['ip']
                            port = selected_server['port']
                            print(f"\n选择连接服务器: {hostname}:{port}")
                            # 连接新服务器
                            client = ssh_connect(hostname, port, username, password)
                            if not client:
                                print("❌ 连接服务器失败，退出操作")
                                break
                            break
                        else:
                            print("❌ 无效的选择，请重新输入")
                elif choice == '6':
                    # 6. 退出
                    print("\n退出操作")
                    break

                else:
                    print("\n❌ 无效的选择，请重新输入")

                # 询问是否继续
                continue_choice = input("\n是否继续操作？(y/n): ").lower()
                if continue_choice != 'y':
                    break
        finally:
            # 关闭连接
            client.close()
            print("\n✅ 已关闭SSH连接")


if __name__ == "__main__":
    main()