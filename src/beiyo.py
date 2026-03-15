from selenium import webdriver
from selenium.webdriver.edge.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
import time

# 配置Edge驱动路径
driver_path = r"D:\edgedriver\edgedriver_win64\msedgedriver.exe"

# 初始化Edge浏览器
service = Service(driver_path)
driver = webdriver.Edge(service=service)

try:
    # 打开目标网址
    driver.get("http://10.220.42.152:8080")

    # 等待页面加载
    WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.TAG_NAME, "body"))
    )

    # 定义默认的用户名和密码
    username = "admin1"
    password = "tpri@0816"

    # 尝试输入用户名和密码
    # 注意：这里的元素定位器需要根据实际页面结构进行调整
    try:
        # 输入用户名
        username_input = WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.NAME, "username"))
        )
        username_input.send_keys(username)

        # 输入密码
        password_input = WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.NAME, "password"))
        )
        password_input.send_keys(password)

        # 点击登录按钮
        login_button = WebDriverWait(driver, 10).until(
            EC.element_to_be_clickable((By.XPATH, "//button[@type='submit']"))
        )
        login_button.click()

        # 等待登录结果
        time.sleep(3)
        print("登录操作完成")

    except Exception as e:
        print(f"元素定位失败: {e}")
        print("请根据实际页面结构调整元素定位器")

    # 等待一段时间以便查看结果
    time.sleep(5)

except Exception as e:
    print(f"测试过程中出现错误: {e}")