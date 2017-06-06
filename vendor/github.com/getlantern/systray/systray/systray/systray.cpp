// systray.cpp : Defines the exported functions for the DLL application.
//

// dllmain.cpp : Defines the entry point for the DLL application.
#include "stdafx.h"
#include "systray.h"

// Message posted into message loop when Notification Icon is clicked
#define WM_SYSTRAY_MESSAGE (WM_USER + 1)

static NOTIFYICONDATA nid;
static HWND hWnd;
static HMENU hTrayMenu;

void (*systray_menu_item_selected)(int menu_id);

void reportWindowsError(const char* action) {
	LPTSTR pErrMsg = NULL;
	DWORD errCode = GetLastError();
	DWORD result = FormatMessage(FORMAT_MESSAGE_ALLOCATE_BUFFER|
			FORMAT_MESSAGE_FROM_SYSTEM|
			FORMAT_MESSAGE_ARGUMENT_ARRAY,
			NULL,
			errCode,
			LANG_NEUTRAL,
			pErrMsg,
			0,
			NULL);
	printf("Systray error %s: %d %s\n", action, errCode, pErrMsg);
}

void ShowMenu(HWND hWnd) {
	POINT p;
	if (0 == GetCursorPos(&p)) {
		reportWindowsError("get tray menu position");
		return;
	};
	SetForegroundWindow(hWnd); // Win32 bug work-around
	TrackPopupMenu(hTrayMenu, TPM_BOTTOMALIGN | TPM_LEFTALIGN, p.x, p.y, 0, hWnd, NULL);

}

int GetMenuItemId(int index) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_DATA;
	if (0 == GetMenuItemInfo(hTrayMenu, index, TRUE, &menuItemInfo)) {
		reportWindowsError("get menu item id");
		return -1;
	}
	return menuItemInfo.dwItemData;
}

LRESULT CALLBACK WndProc(HWND hWnd, UINT message, WPARAM wParam, LPARAM lParam) {
	switch (message) {
		case WM_MENUCOMMAND:
			{
				int menuId = GetMenuItemId(wParam);
				if (menuId != -1) {
					systray_menu_item_selected(menuId);
				}
			}
			break;
		case WM_DESTROY:
			PostQuitMessage(0);
			break;
		case WM_SYSTRAY_MESSAGE:
			switch(lParam) {
				case WM_RBUTTONUP:
					ShowMenu(hWnd);
					break;
				case WM_LBUTTONUP:
					ShowMenu(hWnd);
					break;
				default:
					return DefWindowProc(hWnd, message, wParam, lParam);
			};
			break;
		default:
			return DefWindowProc(hWnd, message, wParam, lParam);
	}
	return 0;
}

void MyRegisterClass(HINSTANCE hInstance, TCHAR* szWindowClass) {
	WNDCLASSEX wcex;

	wcex.cbSize = sizeof(WNDCLASSEX);
	wcex.style          = CS_HREDRAW | CS_VREDRAW;
	wcex.lpfnWndProc    = WndProc;
	wcex.cbClsExtra     = 0;
	wcex.cbWndExtra     = 0;
	wcex.hInstance      = hInstance;
	wcex.hIcon          = LoadIcon(NULL, IDI_APPLICATION);
	wcex.hCursor        = LoadCursor(NULL, IDC_ARROW);
	wcex.hbrBackground  = (HBRUSH)(COLOR_WINDOW+1);
	wcex.lpszMenuName   = 0;
	wcex.lpszClassName  = szWindowClass;
	wcex.hIconSm        = LoadIcon(NULL, IDI_APPLICATION);

	RegisterClassEx(&wcex);
}

HWND InitInstance(HINSTANCE hInstance, int nCmdShow, TCHAR* szWindowClass) {
	HWND hWnd = CreateWindow(szWindowClass, TEXT(""), WS_OVERLAPPEDWINDOW,
			CW_USEDEFAULT, 0, CW_USEDEFAULT, 0, NULL, NULL, hInstance, NULL);
	if (!hWnd) {
		return 0;
	}

	ShowWindow(hWnd, nCmdShow);
	UpdateWindow(hWnd);

	return hWnd;
}


BOOL createMenu() {
	hTrayMenu = CreatePopupMenu();
	MENUINFO menuInfo;
	menuInfo.cbSize = sizeof(MENUINFO);
	menuInfo.fMask = MIM_APPLYTOSUBMENUS | MIM_STYLE;
	menuInfo.dwStyle = MNS_NOTIFYBYPOS;
	return SetMenuInfo(hTrayMenu, &menuInfo);
}

BOOL addNotifyIcon() {
	nid.cbSize = sizeof(NOTIFYICONDATA);
	nid.hWnd = hWnd;
	nid.uID = 100;
	nid.uCallbackMessage = WM_SYSTRAY_MESSAGE;
	nid.uFlags = NIF_MESSAGE;
	return Shell_NotifyIcon(NIM_ADD, &nid);
}

int nativeLoop(void (*systray_ready)(int ignored), void (*_systray_menu_item_selected)(int menu_id)) {
	systray_menu_item_selected = _systray_menu_item_selected;

	HINSTANCE hInstance = GetModuleHandle(NULL);
	TCHAR* szWindowClass = TEXT("SystrayClass");
	MyRegisterClass(hInstance, szWindowClass);
	hWnd = InitInstance(hInstance, FALSE, szWindowClass); // Don't show window
	if (!hWnd) {
		return EXIT_FAILURE;
	}
	if (!createMenu() || !addNotifyIcon()) {
		return EXIT_FAILURE;
	}
	systray_ready(0);

	MSG msg;
	while (GetMessage(&msg, NULL, 0, 0)) {
		TranslateMessage(&msg);
		DispatchMessage(&msg);
	}   
	return EXIT_SUCCESS;
}


void setIcon(const wchar_t* iconFile) {
	HICON hIcon = (HICON) LoadImage(NULL, iconFile, IMAGE_ICON, 64, 64, LR_LOADFROMFILE);
	if (hIcon == NULL) {
		reportWindowsError("load icon image");
	} else {
		nid.hIcon = hIcon;
		nid.uFlags = NIF_ICON;
		Shell_NotifyIcon(NIM_MODIFY, &nid);
	}
}

void setTooltip(const wchar_t* tooltip) {
	wcsncpy_s(nid.szTip, tooltip, sizeof(nid.szTip)/sizeof(wchar_t));
	nid.uFlags = NIF_TIP;
	Shell_NotifyIcon(NIM_MODIFY, &nid);
}

void add_or_update_menu_item(int menuId, wchar_t* title, wchar_t* tooltip, short disabled, short checked) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_FTYPE | MIIM_STRING | MIIM_DATA | MIIM_STATE;
	menuItemInfo.fType = MFT_STRING;
	menuItemInfo.dwTypeData = title;
	menuItemInfo.cch = wcslen(title) + 1;
	menuItemInfo.dwItemData = (ULONG_PTR)menuId;
	menuItemInfo.fState = 0;
	if (disabled == 1) {
		menuItemInfo.fState |= MFS_DISABLED;
	}
	if (checked == 1) {
		menuItemInfo.fState |= MFS_CHECKED;
	}

	int itemCount = GetMenuItemCount(hTrayMenu);
	int i;
	for (i = 0; i < itemCount; i++) {
		int id = GetMenuItemId(i);
		if (-1 == id) {
			continue;
		}
		if (menuId == id) {
			SetMenuItemInfo(hTrayMenu, i, TRUE, &menuItemInfo);
			break;
		}
	}
	if (i == itemCount) {
		InsertMenuItem(hTrayMenu, -1, TRUE, &menuItemInfo);
	}
}

void quit() {
	Shell_NotifyIcon(NIM_DELETE, &nid);
}
