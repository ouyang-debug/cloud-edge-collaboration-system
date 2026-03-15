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


# ====================== 后续操作函数 ======================
def navigate_to_environment_definition(driver):
    """导航到环境定义页面"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)
        
        # 1. 点击云边协同菜单
        logger.info("开始点击云边协同菜单")
        cloud_edge_menu = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//span[text()='云边协同']"))
        )
        cloud_edge_menu.click()
        logger.info("✅ 云边协同菜单点击成功")
        
        # 2. 点击环境管理
        logger.info("开始点击环境管理")
        environment_management = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//span[text()='环境管理']"))
        )
        environment_management.click()
        logger.info("✅ 环境管理点击成功")
        
        # 3. 点击环境定义
        logger.info("开始点击环境定义")
        environment_definition = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//span[text()='环境定义']"))
        )
        environment_definition.click()
        logger.info("✅ 环境定义点击成功")
        
        # 4. 等待环境定义页面加载完成
        wait.until(EC.presence_of_element_located((By.XPATH, "//button[contains(text(), '新增环境')]")))
        logger.info("✅ 环境定义页面加载完成")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 导航到环境定义页面失败：{str(e)}")
        # 截图保存以便调试
        screenshot_path = f"navigate_error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存导航错误截图：{screenshot_path}")
        return False

def maintain_environment_info(driver):
    """维护环境信息 - 新增环境定义"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)
        
        logger.info("开始维护环境信息 - 新增环境定义")
        
        # 1. 点击新增环境按钮
        logger.info("开始点击新增环境按钮")
        add_button = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//button[contains(text(), '新增环境')]"))
        )
        add_button.click()
        logger.info("✅ 点击新增环境按钮成功")
        
        # 2. 等待表单弹窗加载
        wait.until(EC.presence_of_element_located((By.XPATH, "//div[contains(@class, 'dialog') or contains(@class, 'modal')]")))
        logger.info("✅ 表单弹窗加载完成")
        
        # 3. 填写环境编码
        logger.info("开始填写环境编码")
        environment_code = wait.until(
            EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入环境编码']"))
        )
        environment_code.clear()
        timestamp = str(int(time.time()))
        environment_code.send_keys("TEST" + timestamp)
        logger.info("✅ 填写环境编码成功")
        
        # 4. 填写环境名称
        logger.info("开始填写环境名称")
        environment_name = wait.until(
            EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入环境名称']"))
        )
        environment_name.clear()
        environment_name.send_keys("测试环境" + timestamp)
        logger.info("✅ 填写环境名称成功")
        
        # 5. 填写环境描述
        logger.info("开始填写环境描述")
        environment_desc = wait.until(
            EC.presence_of_element_located((By.XPATH, "//textarea[@placeholder='请输入内容']"))
        )
        environment_desc.clear()
        environment_desc.send_keys("自动化测试环境 - " + time.strftime("%Y-%m-%d %H:%M:%S"))
        logger.info("✅ 填写环境描述成功")
        
        # 6. 填写备注
        logger.info("开始填写备注")
        remark = wait.until(
            EC.presence_of_element_located((By.XPATH, "//textarea[contains(@placeholder, '备注')]"))
        )
        remark.clear()
        remark.send_keys("自动化测试创建")
        logger.info("✅ 填写备注成功")
        
        # 7. 显示顺序保持默认值1
        logger.info("显示顺序保持默认值1")
        
        # 8. 状态保持默认启用
        logger.info("状态保持默认启用")
        
        # 9. 点击确认按钮
        logger.info("开始点击确认按钮")
        confirm_button = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//button[contains(text(), '确认')]"))
        )
        confirm_button.click()
        logger.info("✅ 点击确认按钮成功")
        
        # 10. 等待保存成功提示
        try:
            wait.until(EC.presence_of_element_located((By.XPATH, "//div[contains(text(), '成功')]")))
            logger.info("✅ 环境定义保存成功")
        except Exception as e:
            logger.warning(f"⚠️ 保存成功提示未找到：{str(e)}")
        
        # 11. 等待弹窗关闭
        wait.until(EC.invisibility_of_element_located((By.XPATH, "//div[contains(@class, 'dialog') or contains(@class, 'modal')]")))
        logger.info("✅ 表单弹窗关闭成功")
        
        # 12. 等待页面刷新
        time.sleep(2)
        logger.info("✅ 环境定义新增完成")
        return True
        
    except Exception as e:
        logger.error(f"❌ 维护环境信息失败：{str(e)}")
        # 截图保存以便调试
        screenshot_path = f"error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        return False


# ====================== 主函数 ======================
if __name__ == "__main__":
    # 执行登录
    driver = login_with_retry()

    # 登录成功后的后续操作
    if driver:
        try:
            # 导航到环境定义页面
            if navigate_to_environment_definition(driver):
                # 维护环境信息
                maintain_environment_info(driver)
            
            # 操作完成后停留一段时间观察结果
            time.sleep(500)
            logger.info("🔚 自动化流程完成，关闭浏览器")
        finally:
            driver.quit()
    else:
        logger.error("🔚 自动化流程终止：登录失败")