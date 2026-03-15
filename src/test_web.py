import logging
import time
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.edge.service import Service
from selenium.common.exceptions import (
    NoSuchElementException,
    ElementClickInterceptedException,
    TimeoutException,
    WebDriverException
)

# ====================== 日志配置 ======================
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('login_automation.log', encoding='utf-8'),  # 日志文件
        logging.StreamHandler()  # 控制台输出
    ]
)
logger = logging.getLogger(__name__)

# ====================== 核心配置 ======================
DRIVER_PATH = r'D:\edgedriver\edgedriver_win64\msedgedriver.exe'  # 替换成你的驱动路径
LOGIN_URL = 'http://10.220.42.152:8080'  # 替换成实际登录页URL
USERNAME = 'admin'  # 用户名
PASSWORD = ''  # 密码
MAX_RETRY = 3  # 最大重试次数
WAIT_TIME = 15  # 元素等待时间


# ====================== 登录函数 ======================
def login_with_retry():
    driver = None
    retry_count = 0

    while retry_count < MAX_RETRY:
        try:
            # 初始化浏览器
            service = Service(executable_path=DRIVER_PATH)
            driver = webdriver.Edge(service=service)
            driver.set_page_load_timeout(30)  # 页面加载超时
            driver.maximize_window()
            logger.info(f"第 {retry_count + 1} 次尝试登录，打开页面：{LOGIN_URL}")

            # 打开登录页
            driver.get(LOGIN_URL)
            wait = WebDriverWait(driver, WAIT_TIME)

            # 1. 输入用户名
            username_input = wait.until(
                EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入用户名']"))
            )
            username_input.clear()
            username_input.send_keys(USERNAME)
            logger.info("✅ 用户名输入完成")

            # 2. 输入密码
            password_input = wait.until(
                EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入密码']"))
            )
            password_input.send_keys(PASSWORD)
            logger.info("✅ 密码输入完成")

            # 3. 点击登录按钮（带重试的点击逻辑）
            login_btn = wait.until(
                EC.element_to_be_clickable((By.CSS_SELECTOR, "button.login-btn"))
            )
            login_btn.click()
            logger.info("✅ 登录按钮点击完成")

            # 4. 验证登录成功（两种判断方式，选其一）
            # 方式1：判断URL变化
            wait.until(EC.url_changes(LOGIN_URL))
            # 方式2：判断首页特征元素（替换成实际首页元素）
            # wait.until(EC.presence_of_element_located((By.XPATH, "//h1[text()='首页']")))

            logger.info("🎉 登录成功！当前URL：%s", driver.current_url)
            return driver  # 返回driver供后续操作

        except TimeoutException:
            retry_count += 1
            logger.error(f"❌ 元素加载超时，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)  # 重试前等待

        except (NoSuchElementException, ElementClickInterceptedException):
            retry_count += 1
            logger.error(f"❌ 元素定位/点击失败，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

        except WebDriverException as e:
            retry_count += 1
            logger.error(f"❌ 浏览器驱动异常：{str(e)}，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

        except Exception as e:
            retry_count += 1
            logger.error(f"❌ 未知异常：{str(e)}，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

    logger.critical("❌ 登录失败：已重试 %d 次仍未成功", MAX_RETRY)
    if driver:
        driver.quit()
    return None


# ====================== 主函数 ======================
if __name__ == "__main__":
    # 执行登录
    driver = login_with_retry()

    # 登录成功后的后续操作（示例）
    if driver:
        try:
            # 示例：停留10秒后关闭
            time.sleep(10)
            logger.info("🔚 自动化流程完成，关闭浏览器")
        finally:
            driver.quit()
    else:
        logger.error("🔚 自动化流程终止：登录失败")