#!/usr/bin/python3

import requests
import json
import os
import argparse
import zipfile
import io


# Define the command-line arguments
default_api_token = os.environ.get("API_TOKEN","")
default_api_endpoint = os.environ.get("API_ENDPOINT","")
parser = argparse.ArgumentParser(description='ScrapeUI:\nSimple frontend for wakizaki')
parser.add_argument('--url', action='store', help='API URL of wakizaki server, optional', default = default_api_endpoint )
parser.add_argument('--token', action='store', help='API token for wakizaki server', default = default_api_token)
parser.add_argument('--test', action='store_true', help='Send requests to wakizaki test server')
parser.add_argument('--browser', action='store_true', help='Use browser to scrape pages')
parser.add_argument('--output', type=str, help='output file path')
parser.add_argument('--index', action='store_true', help='force download only the index file')

parser.add_argument('action', choices=['create', 'add', 'list', 'stats', 'download', 'delete', 'scrape_all'], help='Action to perform')
parser.add_argument('--name', help='task name')
parser.add_argument('--id', type=str, help='task id')
parser.add_argument('--url_list', type=str, help='file path containing url list')


class Task:
    def __init__(self, taskName: str):
        self.taskName = taskName
        self.taskID = createTask(taskName)

    def addURL(self, pageList: list) -> bool:
        return addURL(self.taskID, pageList)

    def getStat(self) -> dict:
        return getTaskStat(self.taskID)

class Page:
    def __init__(self, line: str):
        global browser
        parts = line.split(',', 1)
        if len(parts) == 1:
            self.url = parts[0].strip()
            self.note = ''
        else:
            self.url = parts[0].strip()
            self.note = parts[1].strip()
        self.browser = browser


from json import JSONEncoder
class DictEncoder(JSONEncoder):
    def default(self, o):
        return o.__dict__ 

"""Create a task in the scraper"""
def createTask(taskName: str) -> str:
    payload = json.dumps({"name": taskName, "pages": []})
    response = requests.request("POST", url+"task", headers=headers, data=payload)
    print(response.status_code)
    taskID = response.json()['taskID']
    return taskID


"""Add a list of URL to the task"""
def addURL(taskID: str, pageList: list) -> bool:
    payload = json.dumps({"taskID": taskID, "pages": pageList}, cls=DictEncoder)
    response = requests.request("PUT", url+"task/"+taskID, headers=headers, data=payload)

    if response.status_code == 200:
        return True
    else:
        return False


"""List scrape tasks"""
def listTasks() -> list:
    response = requests.request("GET", url+"task", headers=headers)
    taskList = response.json()
    return taskList


"""Get task statistics"""
def getTaskStat(taskID: str) -> dict:
    response = requests.request("GET", url+"task/"+taskID+"/statistics", headers=headers)
    taskStat = response.json()
    return taskStat


"""Download pages"""
def downloadPagesZip(taskID: str) -> bytes:
    response = requests.request("GET", url+"task/"+taskID, headers=headers)

    if response.status_code == 200:
        return response.content
    else:
        return bytes()

"""Download pages index"""
def downloadPagesIndex(taskID: str) -> dict:
    response = requests.request("GET", url+"task/"+taskID+"?indexOnly=true", headers=headers)

    if response.status_code == 200:
        # Unzip the response to get the index file
        try:
            with zipfile.ZipFile(io.BytesIO(response.content)) as z:
                with z.open("index.json") as f:
                    return json.load(f)
        except json.JSONDecodeError:
            print("Error decoding JSON")
            return {}
    else:
        return {}


"""Delete tasks"""
def deleteTask(taskID: str) -> bool:
    response = requests.request("DELETE", url+"task/"+taskID, headers=headers)

    if response.status_code == 200:
        return True
    else:
        return False


"""
Download all the links by extracting from .txt of each folder.
Folders listed inside scraped-list.txt will be skipped.
"""
def scrape_all():

    taskList = list()
    
    dirList = [i for i in os.listdir() if os.path.isdir(i) and not i.startswith(".")]
    
    with open("./scraped-list.txt", "r") as f:
        scrapedList = f.readlines()
        for i in scrapedList:
            dirName = i.split('#')[0].strip()

            if dirName in dirList:
                dirList.remove(dirName)

    with open("./scraped-list.txt", "a") as f:
        for i in dirList:
            task = Task(i)
            taskList.append(task)
            print("Task created: "+i)
            print("TaskID: " + task.taskID)
            f.write(f"{i} # {task.taskID}\n")

            files = os.listdir(i)

            for j in files:
                if not j.endswith(".txt"):
                    continue

                with open(os.path.join(i, j), "r") as g:
                    urlList = g.readlines()
                    urlList = list(filter(lambda s:s != "", [k.strip() for k in urlList]))
                    pageList = list(map(Page, urlList))
                    task.addURL(pageList)

def main(args):
    if args.test:
        global url
        url = "http://9.134.211.18:3033/"
        global token
        token = "devtesttoken"

    if args.action == "create":
        if args.name == None:
            print("Please specify a task name")
            return

        taskid = createTask(args.name)
        print(f"Task created: {args.name}, TaskID: {taskid}")

    elif args.action == "add":
        taskid = args.id
        
        if taskid == None:
            print("Please specify a task id")
            return

        with open(args.url_list, "r") as f:
            urlList = f.readlines()
            urlList = list(filter(lambda s:s != "", [k.strip() for k in urlList]))
            pageList = list(map(Page, urlList))
            addURL(taskid, pageList)
            print(f"{len(urlList)} URLs added to task {taskid}")

    elif args.action == "list":
        output = listTasks()
        print(json.dumps(output, indent=4))

    elif args.action == "stats":
        taskid = args.id
        
        if taskid == None:
            print("Please specify a task id")
            return

        stats = getTaskStat(taskid)
        print(json.dumps(stats, indent=4))

    elif args.action == "download":
        taskid = args.id
        if taskid == None:
            print("Please specify a task id")
            return
        
        filename = ""
        if args.output:
            filename = args.output

        # If downloading index
        if args.index:
            output = downloadPagesIndex(taskid)

            if filename == "":
                filename = taskid + ".json"
            
            with open(filename, "w") as f:
                f.write(json.dumps(output, indent=4))

            print(f"Downloaded index {filename}")

        # if downloading zip
        else:
            output = downloadPagesZip(taskid)

            if filename == "":
                filename = taskid + ".zip"

            with open(filename, "wb") as f:
                f.write(output)

            print(f"Downloaded {filename}")

    elif args.action == "delete":
        taskid = args.id
        
        if taskid == None:
            print("Please specify a task id")
            return

        if deleteTask(taskid):
            print(f"Task {taskid} deleted")
        else:
            print(f"Error deleting task {taskid}")

    elif args.action == "scrape_all":
        scrape_all()
    

# Parse the command-line arguments and call the main function
if __name__ == '__main__':
    args = parser.parse_args()

    global url 
    url = args.url
    global token
    token = args.token
    global browser
    browser = args.browser

    headers = {
        'Content-Type': 'application/json',
        'Token': token
    }

    main(args)
