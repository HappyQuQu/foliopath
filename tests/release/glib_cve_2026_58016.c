#include <gio/gio.h>

int
main (void)
{
  static const char *const invalid_xml[] = {
    "<node><interface name=\"I\"><method name=\"M\"><node/></method></interface></node>",
    "<node><interface name=\"I\"><signal name=\"S\"><node/></signal></interface></node>",
    "<node><interface name=\"I\"><property name=\"P\" type=\"s\" access=\"read\"><node/></property></interface></node>",
    "<node><interface name=\"I\"><method name=\"M\"><arg type=\"s\"><node/></arg></method></interface></node>",
  };

  for (gsize i = 0; i < G_N_ELEMENTS (invalid_xml); i++)
    {
      GError *error = NULL;
      GDBusNodeInfo *node = g_dbus_node_info_new_for_xml (invalid_xml[i], &error);

      if (node != NULL ||
          error == NULL ||
          error->domain != G_MARKUP_ERROR ||
          error->code != G_MARKUP_ERROR_INVALID_CONTENT)
        {
          if (node != NULL)
            g_dbus_node_info_unref (node);
          g_clear_error (&error);
          return 1;
        }

      g_clear_error (&error);
    }

  return 0;
}
